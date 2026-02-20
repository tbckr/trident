# Product Requirements Document (PRD): Trident

## 1. Introduction

**Trident** is a port and evolution of the Python-based OSINT tool [Harpoon](https://github.com/Te-k/harpoon), rewritten in **Go**. The goal is to create a high-performance, statically compiled CLI tool for Open Source Intelligence gathering. It automates querying various threat intelligence, network, and identity/social media platforms.

This document defines a phased delivery model with a deliberately lean MVP (Phase 1) to validate the architecture and deliver immediate value, followed by incremental capability phases.

## 2. Goals & Constraints

* **Language:** Go (Golang) — single binary, cross-platform distribution.
* **Architecture:** Modular plugin design where each service implements a common interface.
* **Minimal Dependencies:** No third-party SDKs for service integrations. All API interactions are implemented natively via HTTP client.
* **Configuration:** Centralized config file (`~/.config/trident/config.yaml`) managed via Viper.

### Non-Goals (all phases)

* **GUI:** No graphical user interface; purely CLI-based.
* **VDR/VEX:** Automatic generation of Vulnerability Disclosure Reports or VEX documents (exploration in Phase 5+).

---

## 3. Phase Overview

| Phase | Name | Focus | Services |
|-------|------|-------|----------|
| **1** | **MVP** | Core framework + 3 keyless services, basic output, CI | DNS, ASN, crt.sh |
| **2** | **Bulk & OpSec Foundations** | Stdin processing, concurrency, proxy, PAP, defanging, remaining keyless services | + ThreatMiner, PGP |
| **3** | **Release Hardening** | Supply chain security, release automation, rate limiting, jitter | — |
| **4** | **Passive Expansion** | Additional keyless/low-friction services, caching | + cache, quad9, tor, robtex, umbrella |
| **5** | **API-Key Services** | Services requiring authentication | Shodan, Censys, VirusTotal, etc. |
| **6+** | **Advanced OpSec & Research** | TLS evasion, honeypot detection, encrypted workspace, behavioral mimicry | — |

---

## 4. Phase 1 — MVP

### 4.1 Goal

Deliver a working CLI tool that an analyst can install and immediately use for basic domain/IP reconnaissance — no API keys, no complex setup.

### 4.2 User Stories

* **US-1:** As an analyst, I want to resolve domain names and retrieve DNS records, so I can start an investigation immediately.
* **US-2:** As an analyst, I want to retrieve ASN information for an IP address, so I can identify the hosting provider.
* **US-3:** As an investigator, I want to find subdomains for a target domain using certificate transparency logs (crt.sh).
* **US-4:** As a user, I want a unified CLI command structure that is consistent across different services.

### 4.3 Technical Stack

| Component | Choice | Notes |
|-----------|--------|-------|
| CLI Framework | [spf13/cobra](https://github.com/spf13/cobra) | Command structure, flag parsing, help generation |
| Configuration | [spf13/viper](https://github.com/spf13/viper) | YAML config + env variable support |
| HTTP Client | [imroc/req](https://github.com/imroc/req) (v3) | All HTTP interactions; no external SDKs |
| Logging | [log/slog](https://pkg.go.dev/log/slog) | Standard library only, no zap/logrus |
| Table Output | [olekukonko/tablewriter](https://github.com/olekukonko/tablewriter) | ASCII table formatting |
| Testing | [stretchr/testify](https://github.com/stretchr/testify) + [jarcoal/httpmock](https://github.com/jarcoal/httpmock) | Assertions, mocking, HTTP transport mocking |

### 4.4 Services

#### 4.4.1 DNS (`dns`)

* **Input:** Domain name or IP address.
* **Output:** A, AAAA, MX, NS, TXT records.
* **Implementation:** Go's native `net` package (no HTTP required).

#### 4.4.2 ASN (`asn`)

* **Input:** ASN string (e.g., `AS15169`) or IP address.
* **Output:** AS Description, Country, Registry, allocated prefixes.
* **Implementation:** Query Team Cymru via DNS TXT records using Go's native `net.Resolver`. Format: `<reversed-ip>.origin.asn.cymru.com`. **No** `os/exec` calls to `dig`.

#### 4.4.3 crt.sh (`crtsh`)

* **Input:** Domain name (e.g., `example.com`).
* **Output:** List of subdomains, certificate issuance dates, CAs.
* **Implementation:** HTTP GET to `https://crt.sh/?q=%.<domain>&output=json` via `imroc/req`.

### 4.5 Global Flags (MVP)

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to config file | `~/.config/trident/config.yaml` |
| `--verbose` (`-v`) | Sets slog level to Debug | Info |
| `--output` (`-o`) | Output format: `text`, `json` | `text` |

### 4.6 Architecture Requirements

* **Dependency Injection:** Constructor injection for all services. No global state or singletons.
  * Example: `NewCrtshService(client HttpClientInterface, logger *slog.Logger) *CrtshService`
* **Interfaces:** Define Go interfaces for external interactions (HTTP, DNS resolver) to enable mocking.
* **Test Coverage:** Minimum **80%** code coverage. Integration tests use recorded responses/fixtures.
* **`run` Function Pattern:** `main()` delegates to a `run()` function that accepts dependencies and returns errors, enabling testability.

### 4.7 Input Validation (MVP)

* **Domains:** Conform to standard hostname RFCs (no shell characters, correct TLD format).
* **IPs:** Valid IPv4 or IPv6 (`net.ParseIP`).
* **ASNs:** Must match `^AS\d+$`.
* Reject all other input. No command injection or path traversal possible.

### 4.8 Security (MVP)

* **No Plaintext Secrets:** API keys never printed to stdout/stderr/logs. Masked in debug output.
* **File Permissions:** Config file created with `0600`.
* **Enforced HTTPS:** All HTTP requests use HTTPS. No `InsecureSkipVerify`.
* **Output Sanitization:** Sanitize external data before terminal output (prevent ANSI escape injection).
* **Env Variable Precedence:** Viper prefers env vars (e.g., `TRIDENT_*`) over config file values.

### 4.9 CI/CD (MVP)

* **GitHub Actions:** Linting (`golangci-lint`, strict) + Testing on push/PR.
* **Versioning:** Semantic Versioning (SemVer).
* Build must fail on any lint error or test failure.

### 4.10 Compatibility

* Must compile and run on **Linux**, **macOS**, **Windows**.
* Use `filepath.Join` for path handling.
* Respect OS config standards (XDG on Linux, AppData on Windows) with graceful fallback.

### 4.11 Explicitly NOT in MVP

The following are deferred to later phases to keep the MVP lean:

* ❌ Stdin/bulk input processing
* ❌ Concurrency / worker pools / `--concurrency` flag
* ❌ Proxy support (`--proxy`)
* ❌ User-Agent spoofing
* ❌ PAP system (`--pap-limit`)
* ❌ Output defanging (`--defang` / `--no-defang`)
* ❌ Plain output mode (`-o plain`)
* ❌ Rate limiting / jitter
* ❌ GoReleaser / SBOM / Cosign
* ❌ `burn` / `wipe` command
* ❌ ThreatMiner, PGP services
* ❌ `gosec` / `govulncheck` in CI
* ❌ Renovate

---

## 5. Phase 2 — Bulk & OpSec Foundations

### 5.1 Goal

Enable power-user workflows (bulk processing, piping) and lay the OpSec foundation.

### 5.2 User Stories

* **US-5:** As a power user, I want to pipe a list of domains/IPs via stdin for bulk analysis.
* **US-6:** As an investigator, I want to mask my source IP via a proxy to avoid detection.
* **US-7:** As an investigator, I want to set a PAP limit to prevent accidental active interaction.
* **US-8:** As an analyst, I want output auto-defanging under strict OpSec rules.
* **US-9:** As a power user, I want `--no-defang` to enable piping raw results.
* **US-10:** As a user, I want to control concurrency to optimize performance or reduce load.
* **US-11:** As a power user, I want a `plain` output mode for piping into other tools.

### 5.3 New Services

* **ThreatMiner** (`threatminer`) — Contextual threat intelligence. Input: Domain, IP, Hash. API: `https://api.threatminer.org/v2/`.
* **PGP** (`pgp`) — PGP key search. Input: Email or Name. API: HKP servers (`https://keys.openpgp.org`).

### 5.4 New Capabilities

| Capability | Details |
|------------|---------|
| **Stdin Processing** | Dual input mode (args OR stdin). One entry per line, trim whitespace/empty lines. |
| **Worker Pools** | Goroutine-based worker pool, size via `--concurrency` (default: 10). Bounded channels. |
| **Proxy Support** | `--proxy` flag. HTTP, HTTPS, SOCKS5. |
| **User-Agent Spoofing** | Rotating modern browser UA strings by default. `--user-agent` override. |
| **PAP System** | Command classification (RED/AMBER/GREEN/WHITE). `--pap-limit` enforcement. Visual indicator in help/logs. |
| **Defanging** | Auto-enabled at PAP AMBER/RED. `--defang` / `--no-defang` flags. Domains, IPs, URLs. JSON stays raw unless `--defang` explicit. |
| **Plain Output** | `-o plain` — one result per line for piping. |

### 5.5 New Global Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--proxy` | Proxy URL (HTTP/HTTPS/SOCKS5) | — |
| `--user-agent` | Override User-Agent string | rotating browser UAs |
| `--pap-limit` | Permissible Actions Protocol limit | `white` |
| `--defang` | Force output defanging | — |
| `--no-defang` | Disable output defanging | — |
| `--concurrency` (`-c`) | Worker pool size | 10 |

### 5.6 PAP Classification

| Service | PAP Level |
|---------|-----------|
| DNS | **GREEN** |
| ASN | **AMBER** |
| crt.sh | **AMBER** |
| ThreatMiner | **AMBER** |
| PGP | **AMBER** |

---

## 6. Phase 3 — Release Hardening

### 6.1 Goal

Production-grade release pipeline with supply chain security, rate limiting, and traffic analysis protection.

### 6.2 Capabilities

| Capability | Details |
|------------|---------|
| **GoReleaser** | Build, package, publish releases to GitHub. Cross-platform binaries. |
| **SBOM** | CycloneDX generation per release artifact (via GoReleaser). |
| **Cosign Signing** | Signed binaries and checksums for integrity/authenticity. |
| **Reproducible Builds** | `-trimpath` flags, deterministic build config. |
| **Rate Limiting** | Token Bucket algorithm (`golang.org/x/time/rate`) per service. Respect `X-RateLimit-*` headers and HTTP 429. |
| **Request Jitter** | ±20% random variation on request intervals. |
| **DNS Leak Prevention** | Remote DNS resolution when using SOCKS5 proxy. |
| **gosec** | SAST scanning in CI. Use in conjunction with golangci-lint. |
| **govulncheck** | SCA scanning in CI. |
---

## 7. Appendix: Complete Service Matrix

| Service | API Key? | PAP Level | Phase |
|---------|----------|-----------|-------|
| **dns** | No | GREEN | 1 |
| **asn** | No | AMBER | 1 |
| **crtsh** | No | AMBER | 1 |
| **threatminer** | No | AMBER | 2 |
| **pgp** | No | AMBER | 2 |
| **cache** | No (mostly) | AMBER | 4 |
| **quad9** | No (DNS) | AMBER | 4 |
| **tor** | No | AMBER | 4 |
| **robtex** | No (limited) | AMBER | 4 |
| **umbrella** | No (CSV) | RED | 4 |
| **shodan** | Yes | AMBER | 5 |
| **vt** | Yes | AMBER | 5 |
| **censys** | Yes | AMBER | 5 |
| **greynoise** | Yes | AMBER | 5 |
| **securitytrails** | Yes | AMBER | 5 |
| **hibp** | Yes | AMBER | 5 |
| **otx** | Yes | AMBER | 5 |
| **ipinfo** | Yes | AMBER | 5 |
| **hunter** | Yes | AMBER | 5 |
| **urlscan** | Yes | GREEN | 5 |
| **github** | Yes | AMBER | 5 |
| **pulsedive** | Yes | AMBER | 5 |
| **pt** | Yes | AMBER | 5 |
| **binaryedge** | Yes | AMBER | 5 |
| **circl** | Yes | AMBER | 5 |
| **fullcontact** | Yes | AMBER | 5 |
| **hybrid** | Yes | AMBER | 5 |
| **koodous** | Yes | AMBER | 5 |
| **malshare** | Yes | AMBER | 5 |
| **misp** | Yes | AMBER | 5 |
| **numverify** | Yes | AMBER | 5 |
| **opencage** | Yes | AMBER | 5 |
| **permacc** | Yes | GREEN | 5 |
| **safebrowsing** | Yes | AMBER | 5 |
| **spyonweb** | Yes | AMBER | 5 |
| **telegram** | Yes | AMBER | 5 |
| **threatcrowd** | Yes | AMBER | 5 |
| **threatgrid** | Yes | AMBER | 5 |
| **totalhash** | Yes | AMBER | 5 |
| **twitter** | Yes | AMBER | 5 |
| **urlhaus** | Yes | AMBER | 5 |
| **xforce** | Yes | AMBER | 5 |
| **zetalytics** | Yes | AMBER | 5 |
| **ip2locationio** | Yes | AMBER | 5 |
| **certspotter** | Yes | AMBER | 5 |
