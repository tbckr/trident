<p align="center">
  <img src="logo.png" alt="trident logo" width="200">
</p>

# trident

[![CI](https://github.com/tbckr/trident/actions/workflows/ci.yml/badge.svg)](https://github.com/tbckr/trident/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/tbckr/trident)](https://github.com/tbckr/trident/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/tbckr/trident)](https://github.com/tbckr/trident/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/tbckr/trident)](https://goreportcard.com/report/github.com/tbckr/trident)
[![License: GPL-3.0](https://img.shields.io/github/license/tbckr/trident)](LICENSE.md)

**Fast, keyless OSINT in a single binary.** DNS lookups, Cymru ASN info, certificate transparency, threat intelligence, PGP key search, and CDN/provider detection — no API keys, no registration, no configuration required.

trident is a Go port and evolution of the Python [Harpoon](https://github.com/Te-k/harpoon) tool, built for analysts and security researchers who live in the terminal.

```console
$ trident dns example.com
+------+------------------------------------------+
| TYPE | VALUE                                    |
+------+------------------------------------------+
| NS   | a.iana-servers.net.                      |
|      | b.iana-servers.net.                      |
+------+------------------------------------------+
| A    | 93.184.216.34                            |
+------+------------------------------------------+
| AAAA | 2606:2800:21f:cb07:6819:42b5:ba16:c9cb  |
+------+------------------------------------------+
| MX   | 0 .                                      |
+------+------------------------------------------+
| TXT  | "v=spf1 -all"                            |
+------+------------------------------------------+
```

---

## Contents

- [Installation](#installation)
- [Verify Release Artifacts](#verify-release-artifacts)
- [Quickstart](#quickstart)
- [Features](#features)
- [Services](#services)
- [Output Formats](#output-formats)
- [Bulk Input](#bulk-input)
- [PAP System](#pap-system)
- [Configuration](#configuration)
- [Global Flags](#global-flags)
- [Commands Reference](#commands-reference)
- [Development](#development)
- [Responsible Use](#responsible-use)
- [Contributing](#contributing)
- [Security](#security)
- [Code of Conduct](#code-of-conduct)

---

## Installation

**The fastest way** — requires Go 1.26+:

```bash
go install github.com/tbckr/trident/cmd/trident@latest
```

**Pre-built binaries** — download for Linux, macOS, or Windows (amd64/arm64) from the [releases page](https://github.com/tbckr/trident/releases). Linux packages (`.deb`, `.rpm`, `.apk`, `pkg.tar.zst`) are included.

**Build from source:**

```bash
git clone https://github.com/tbckr/trident
cd trident
go build -o trident ./cmd/trident
```

---

## Verify Release Artifacts

> **Note:** Starting with **v0.9.0**, SLSA provenance is the only verification method. The previous `cosign sign-blob` signature (`checksums.txt.sigstore.json`) has been removed — SLSA provenance is a strict superset that provides the same trust chain plus structured build metadata. Releases **v0.8.0** and **v0.8.x** support both methods; releases before **v0.8.0** only support `cosign sign-blob` verification (`checksums.txt.sigstore.json`).

Every release is signed with [cosign](https://docs.sigstore.dev/cosign/system_config/installation/)
using keyless signing via GitHub Actions OIDC. The release pipeline produces:

1. **SLSA Provenance v1 attestation** — a structured document proving *how* and *where* the release was built, signed with `cosign attest-blob` → `checksums.txt.slsa-provenance.sigstore.json`
2. **Archive checksums** — every release archive's SHA-256 hash is listed in `checksums.txt`

Full verification chain:

```
cosign verify-blob-attestation  →  provenance attests checksums.txt (build origin + integrity)
sha256sum --check               →  individual archive integrity
```

### Manual verification

```bash
VERSION=v0.9.0
ARCHIVE=trident_Linux_x86_64.tar.gz

# Download verification files
curl -fsSL "https://github.com/tbckr/trident/releases/download/${VERSION}/checksums.txt" -o checksums.txt
curl -fsSL "https://github.com/tbckr/trident/releases/download/${VERSION}/checksums.txt.slsa-provenance.sigstore.json" -o checksums.txt.slsa-provenance.sigstore.json

# Verify SLSA provenance attestation (proves build origin + integrity)
cosign verify-blob-attestation \
  --bundle checksums.txt.slsa-provenance.sigstore.json \
  --type slsaprovenance1 \
  --certificate-identity "https://github.com/tbckr/trident/.github/workflows/release.yml@refs/tags/${VERSION}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  checksums.txt

# Verify the archive checksum (Linux)
sha256sum --check --ignore-missing checksums.txt

# Verify the archive checksum (macOS)
shasum -a 256 --check --ignore-missing checksums.txt
```

### Script

`scripts/verify-release.sh` automates the steps above and works on both Linux and macOS:

```bash
# Download the archive from the releases page first, then:
./scripts/verify-release.sh v0.9.0 trident_Linux_x86_64.tar.gz
```

The script downloads the checksums and SLSA provenance bundle, runs `cosign verify-blob-attestation`, checks the archive hash, and exits non-zero on any failure. It requires `cosign` v2+ and `curl`.

### Checksum-only (without cosign)

If you do not have cosign installed, you can still verify the archive hash against `checksums.txt` after downloading it from the releases page:

```bash
# Linux
sha256sum --check --ignore-missing checksums.txt

# macOS
shasum -a 256 --check --ignore-missing checksums.txt
```

This confirms the archive was not corrupted in transit, but does not verify it was produced by the official release workflow.

---

## Quickstart

```bash
# DNS records — forward lookup or reverse PTR
trident dns example.com
trident dns 8.8.8.8

# ASN info — IP address or ASN number (IPv4 and IPv6)
trident cymru 8.8.8.8
trident cymru AS15169

# Subdomains from certificate transparency logs
trident crtsh example.com

# Threat intelligence — domain, IP, or file hash
trident threatminer example.com
trident threatminer d41d8cd98f00b204e9800998ecf8427e

# PGP key search — by email, name, or fingerprint
trident pgp alice@example.com
trident pgp 0xDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF

# Check whether Quad9 has blocked a domain as malicious
trident quad9 malicious.example.com

# Aggregate DNS recon for an apex domain
trident apex example.com

# Detect CDN, email, and DNS hosting providers via live DNS queries
trident detect example.com

# Identify providers from known DNS record values (no network calls)
trident identify --cname abc.cloudfront.net --mx aspmx.l.google.com --txt "v=spf1 include:_spf.google.com ~all"
```

---

## Features

- **No API keys** — all current services are keyless; install and run immediately
- **Bulk input** — pipe a target list via stdin or pass multiple arguments
- **Three output formats** — `table` (tables), `json`, and `text` (one result per line for piping)
- **PAP system** — Permissible Actions Protocol (RED/AMBER/GREEN/WHITE) prevents accidental active interaction
- **Proxy support** — HTTP, HTTPS, and SOCKS5 proxies; honours `HTTP_PROXY`/`HTTPS_PROXY` env vars automatically
- **Auto-defanging** — URLs and IPs are defanged at strict PAP levels
- **Rate limiting** — per-service token-bucket rate limiter with jitter to avoid detectable request patterns
- **Concurrent processing** — configurable worker pool for fast bulk lookups
- **Cross-platform** — single binary for Linux, macOS, and Windows

---

## Services

| Command | Description | PAP | Data Source |
|---------|-------------|-----|-------------|
| `dns` | A, AAAA, MX, NS, TXT records; reverse PTR | GREEN | Direct DNS resolver |
| `detect` | Detect CDN, email, DNS hosting, and verification providers via live DNS queries (CNAME, MX, NS, TXT) | GREEN | Direct DNS resolver |
| `cymru` | ASN info for IPs and ASN numbers (IPv4 + IPv6) | AMBER | Team Cymru DNS |
| `crtsh` | Subdomain enumeration via certificate transparency | AMBER | [crt.sh](https://crt.sh) |
| `threatminer` | Threat intel for domains, IPs, and file hashes | AMBER | [ThreatMiner](https://www.threatminer.org) |
| `pgp` | PGP key search by email, name, or fingerprint | AMBER | [keys.openpgp.org](https://keys.openpgp.org) |
| `quad9` | Detect whether Quad9 has flagged a domain as malicious | AMBER | [dns.quad9.net](https://www.quad9.net) |
| `apex` | Aggregate DNS recon across many record types and subdomains; CDN/email/DNS/TXT detection and ASN lookup | AMBER | [dns.quad9.net](https://www.quad9.net), Team Cymru DNS |
| `identify` | Identify CDN, email, DNS hosting, and verification providers from known DNS record values (CNAME, MX, NS, TXT) | RED | Local (no network) |

---

## Output Formats

**Table (default)** — formatted ASCII tables for human reading:

```bash
trident dns example.com
trident cymru AS15169 -o table
```

**JSON** — structured output for scripting and integration:

```bash
trident dns example.com -o json
trident crtsh example.com -o json | jq '.subdomains | length'
```

**Text** — one result per line, ideal for piping:

```bash
trident crtsh example.com -o text | sort -u > subdomains.txt
trident dns example.com -o text | grep "^A "
```

---

## Bulk Input

Any command accepts multiple targets as arguments or from stdin (one per line):

```bash
# Multiple arguments
trident dns example.com google.com cloudflare.com

# From a file via stdin
cat targets.txt | trident crtsh

# Combine with other tools
cat /etc/hosts | awk '{print $1}' | trident cymru

# Control concurrency for large lists
cat ips.txt | trident cymru --concurrency=20
```

---

## PAP System

trident implements the [Permissible Actions Protocol (PAP)](https://www.misp-project.org/taxonomies.html#_pap)
to prevent accidental active interaction with targets:

| Level | Meaning | Permitted Services |
|-------|---------|-------------------|
| `red` | Offline/local only — non-detectable | `identify` |
| `amber` | Limited 3rd-party APIs — no direct target contact | `identify` + Cymru, crt.sh, ThreatMiner, PGP, Quad9, apex |
| `green` | Direct target interaction permitted | all AMBER + DNS, `detect` |
| `white` | Unrestricted **(default)** | all |

Set `--pap-limit` to block services above that level:

```bash
# Only use 3rd-party APIs (no direct DNS queries to the target)
trident --pap-limit=amber crtsh example.com

# This will error — AMBER exceeds RED limit
trident --pap-limit=red cymru 8.8.8.8
```

At AMBER and below, URLs and IPs in output are automatically defanged (e.g. `hxxp://`) unless
`--no-defang` is passed.

---

## Configuration

The config file is created automatically at first run:

| Platform | Default Path |
|----------|-------------|
| Linux | `$XDG_CONFIG_HOME/trident/config.yaml` (typically `~/.config/trident/config.yaml`) |
| macOS | `~/Library/Application Support/trident/config.yaml` |
| Windows | `%AppData%\trident\config.yaml` |

Use `trident config set` to modify values without opening the file, or `trident config edit` to
edit directly. The config file supports all global flags plus the `alias` block and
`detect_patterns` section:

```yaml
output: json
pap_limit: amber
concurrency: 20
proxy: socks5://127.0.0.1:9050
detect_patterns:
  url: https://example.com/custom-patterns.yaml  # optional: override download URL
  file: /path/to/patterns.yaml                   # optional: use this file instead of defaults
alias:
  asn: cymru
```

> **Note:** The `alias` block is config-file only — it has no corresponding flag or environment
> variable. Use `trident alias set` / `trident alias delete` to manage aliases, or edit the
> file directly.

> **Note:** When `detect_patterns.file` is not set, trident resolves patterns using the
> following lookup order and uses the first file found:
>
> 1. `<config-dir>/detect.yaml` — user-maintained override
> 2. `<config-dir>/detect-downloaded.yaml` — downloaded via `trident download detect`
> 3. Built-in embedded patterns — always available as the final fallback
>
> Run `trident config path` to find `<config-dir>` on your system.

Environment variables override config file values using the `TRIDENT_` prefix:

| Variable | Corresponding Flag / Key |
|----------|--------------------------|
| `TRIDENT_OUTPUT` | `--output` |
| `TRIDENT_PAP_LIMIT` | `--pap-limit` |
| `TRIDENT_PROXY` | `--proxy` |
| `TRIDENT_USER_AGENT` | `--user-agent` |
| `TRIDENT_CONCURRENCY` | `--concurrency` |
| `TRIDENT_VERBOSE` | `--verbose` |
| `TRIDENT_DEFANG` | `--defang` |
| `TRIDENT_NO_DEFANG` | `--no-defang` |
| `TRIDENT_DETECT_PATTERNS_URL` | `detect_patterns.url` |
| `TRIDENT_DETECT_PATTERNS_FILE` | `--patterns-file` / `detect_patterns.file` |

When `--proxy` / `TRIDENT_PROXY` is not set, trident honours the standard `HTTP_PROXY`,
`HTTPS_PROXY`, and `NO_PROXY` environment variables automatically.

---

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | platform config dir | Config file path |
| `--verbose`, `-v` | `false` | Enable debug logging |
| `--output`, `-o` | `table` | Output format: `table`, `json`, `text` |
| `--concurrency`, `-c` | `10` | Worker pool size for bulk input |
| `--proxy` | — | Proxy URL (`http://`, `https://`, `socks5://`) |
| `--user-agent` | `trident/<version>` | HTTP User-Agent header |
| `--pap-limit` | `white` | PAP limit: `red`, `amber`, `green`, `white` |
| `--defang` | `false` | Force output defanging |
| `--no-defang` | `false` | Disable output defanging |
| `--patterns-file` | — | Custom detect patterns file for `detect`, `apex`, and `identify` |

Use `trident config show` to see the effective configuration.

---

## Commands Reference

### `dns` — DNS Lookups

Resolves A, AAAA, MX, NS, and TXT records for a domain, or performs a reverse PTR lookup for an
IP address. Makes direct queries to the configured DNS resolver (PAP: GREEN).

```bash
trident dns example.com
trident dns 8.8.8.8
trident dns 2001:4860:4860::8888
```

### `cymru` — ASN Lookup

Looks up ASN information for an IP address or ASN number via the Team Cymru DNS service. Supports
both IPv4 and IPv6 (PAP: AMBER).

```bash
trident cymru 8.8.8.8
trident cymru AS15169
trident cymru 2001:4860:4860::8888
```

### `crtsh` — Certificate Transparency

Searches [crt.sh](https://crt.sh) certificate transparency logs for subdomains of a domain
(PAP: AMBER).

```bash
trident crtsh example.com
```

### `threatminer` — Threat Intelligence

Queries the [ThreatMiner](https://www.threatminer.org) API for contextual threat intelligence.
Automatically detects whether input is a domain, IP address, or file hash. Rate-limited to 1
request/second with jitter to avoid triggering ThreatMiner's rate limits (PAP: AMBER).

```bash
trident threatminer example.com
trident threatminer 198.51.100.1
trident threatminer d41d8cd98f00b204e9800998ecf8427e
```

### `pgp` — PGP Key Search

Searches [keys.openpgp.org](https://keys.openpgp.org) for PGP keys by email address, name, or key
fingerprint/ID using the HKP protocol (PAP: AMBER). Fingerprints and key IDs must be prefixed
with `0x`.

```bash
trident pgp alice@example.com
trident pgp "Alice Smith"
trident pgp 0xDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF
```

### `quad9` — Quad9 Threat-Intelligence Check

Detects whether [Quad9](https://www.quad9.net) has flagged a domain as malicious using threat
intelligence from 19+ security partners (PAP: AMBER). Quad9 returns NXDOMAIN with an empty
authority section for known-malicious domains, providing a passive verdict without revealing
the query to the target domain.

```bash
trident quad9 malicious.example.com
trident quad9 example.com malicious.example.com
cat domains.txt | trident quad9
```

### `apex` — Aggregate DNS Recon

Performs parallel DNS reconnaissance for an apex domain via the [Quad9](https://www.quad9.net)
DNS-over-HTTPS resolver (PAP: AMBER). Fans out queries across the apex domain and a large set
of well-known derived hostnames — `www`, `autodiscover`, `mail`, `_dmarc`, `_domainkey`,
`_mta-sts`, `_smtp._tls`, DKIM selectors (`google._domainkey`, `selector1/2._domainkey`),
BIMI (`default._bimi`), and SRV prefixes for SIP and XMPP. Queried record types include A,
AAAA, CAA, CNAME, DNSKEY, HTTPS, MX, NS, SOA, SSHFP, SRV, and TXT.

After gathering records, `apex` runs all four provider detectors:
- **CDN** — from CNAME targets (apex chain, www, and email-security subdomains)
- **Email provider** — from MX records
- **DNS hosting** — from NS records
- **Email provider and verification tokens** — from TXT records across all queried hostnames

Finally, it performs **ASN lookups** (via Team Cymru) for every unique IP found in A/AAAA records.

```bash
trident apex example.com
trident apex example.com example.org
cat domains.txt | trident apex
trident apex --output json example.com
```

### `detect` — Provider Detection

Detects CDN, email, DNS hosting, and domain verification providers for one or more domains by
querying CNAME (apex and www), MX, NS, and TXT records and matching them against known provider
patterns (PAP: GREEN). Unlike `identify`, this command makes live DNS queries to discover the
records.

```bash
trident detect example.com
trident detect example.com google.com
cat domains.txt | trident detect

# Use a custom patterns file for this invocation
trident detect --patterns-file /path/to/patterns.yaml example.com
```

### `identify` — Offline Provider Identification

Matches CNAME, MX, NS, and TXT record values against known provider patterns to identify CDN,
email, DNS hosting, and domain verification providers. Unlike `detect`, no DNS queries are made
— this operates entirely on record values you already have (PAP: RED).

```bash
trident identify --cname abc.cloudfront.net
trident identify --domain example.com --ns ns1.cloudflare.com
trident identify --domain example.com --cname abc.cloudfront.net --mx aspmx.l.google.com --ns ns1.cloudflare.com
trident identify --txt "v=spf1 include:_spf.google.com ~all" --txt "google-site-verification=abc123"

# Use a custom patterns file for this invocation
trident identify --patterns-file /path/to/patterns.yaml --cname abc.cloudfront.net
```

### `download detect` — Update Provider Patterns

Downloads the latest provider detection patterns from a URL and saves them locally. The downloaded
file is stored as `detect-downloaded.yaml` in the config directory and is automatically picked up
by `detect`, `apex`, and `identify` on the next run (PAP: AMBER). A user-maintained
`detect.yaml` in the same directory takes priority over the downloaded file; the built-in embedded
patterns serve as the final fallback when neither file exists. See the
[Configuration](#configuration) section for the full lookup order.

```bash
# Download from the default URL (trident GitHub repository)
trident download detect

# Download from a custom URL
trident download detect --url https://example.com/patterns.yaml

# Save to a custom destination instead of the default config dir
trident download detect --dest /path/to/my-patterns.yaml

# Configure a persistent custom URL
trident config set detect_patterns.url https://example.com/patterns.yaml
trident download detect
```

### `services` — List All Services

Lists every implemented service with its command group, minimum PAP level (MIN PAP), and maximum
PAP level (MAX PAP).

For individual services the two PAP columns are always equal — the service either runs or is
blocked by `--pap-limit`, with no partial behaviour.

For aggregate commands (such as `apex`), the two values may differ: MIN PAP is the lowest PAP
level required to produce any useful output; MAX PAP is the highest level required by any
sub-service. When `--pap-limit` falls between the two, the aggregate command runs but skips the
sub-services whose level exceeds the limit, returning whatever it can gather at that PAP level.

```bash
trident services
trident services -o json
trident services -o text
```

### `config` — Configuration Management

Read and write config file values without opening the file by hand.

| Subcommand | Description |
|------------|-------------|
| `config path` | Print the config file path |
| `config show` | Display all effective config settings |
| `config get <key>` | Print the effective value of a single key |
| `config set <key> <value>` | Write a key–value pair to the config file |
| `config edit` | Open the config file in `$EDITOR` |

```bash
# Print the path to the active config file
trident config path

# Show all effective settings (merged defaults + env vars + file)
trident config show
trident config show -o json

# Read a single setting
trident config get pap_limit

# Persist a setting (hyphens and underscores both accepted)
trident config set output json
trident config set pap-limit amber

# Open the config file in $EDITOR (falls back to vi)
trident config edit
```

**Limitations:**
- `config show` and `config get` report *effective* values — the result of merging built-in
  defaults, `TRIDENT_*` environment variables, and the config file. They do not show what is
  literally written in the file.
- `config set` writes to the file but takes effect on the **next invocation**; the current
  process already loaded config at startup.
- The `aliases` section is not managed by `config set` — use the `alias` subcommand instead.
- Only known configuration keys are accepted (`output`, `pap_limit`, `proxy`, `user_agent`,
  `concurrency`, `verbose`, `defang`, `no_defang`, `detect_patterns.url`, `detect_patterns.file`).

### `alias` — Command Aliases

Define short names that expand to longer command strings. Aliases are stored in the config file
and appear in `trident --help` under *Aliases:*.

```bash
# Create or update an alias
trident alias set asn cymru

# Use the alias — extra arguments are appended after the expansion
trident asn 8.8.8.8

# List all aliases
trident alias list
trident alias list -o json

# Delete an alias
trident alias delete asn
```

**Limitations:**
- Aliases are only expanded when they appear as the **first positional argument**. Running
  `trident --verbose myalias` does **not** trigger expansion because `--verbose` precedes the
  alias name.
- Expansion splits the stored string on whitespace — argument values containing spaces cannot
  be embedded in an alias expansion.
- No shell features — environment variable substitution, pipes, globs, and quoting within
  the expansion string are not interpreted.
- Aliases do not expand recursively; an alias expansion cannot reference another alias.
- Alias names cannot shadow built-in commands (`dns`, `cymru`, `crtsh`, `threatminer`, `pgp`, `quad9`, `detect`, `identify`, `apex`, `services`, `config`, `alias`, `download`, `version`, `completion`).
- Alias names must not start with `-` or contain whitespace.
- Changes take effect on the next invocation.

---

## Development

### Requirements

- Go 1.26+ (`go version`)
- [golangci-lint](https://golangci-lint.run/) v2 (`golangci-lint version`)

### Build & Test

```bash
# Build
go build ./...

# Run all tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# Run tests for a specific service
go test ./internal/services/dns/... -v

# Lint (strict)
golangci-lint run
```

### Project Structure

```
cmd/trident/        # Entry point — delegates to cli.Execute()
internal/
  cli/              # Cobra command tree, global flags, output wiring
  config/           # Viper config loading and flag registration
  httpclient/       # req.Client factory (proxy, UA rotation, debug tracing)
  input/            # Line reader from io.Reader for stdin path
  pap/              # PAP level constants and enforcement
  doh/              # DNS-over-HTTPS client (Quad9 RFC 8484, shared by apex + quad9)
  ratelimit/        # Token-bucket rate limiter with ±20% jitter
  resolver/         # net.Resolver factory with SOCKS5 DNS-leak prevention
  worker/           # Bounded goroutine pool for bulk input
  services/         # One package per OSINT service
    dns/            # DNS record lookups (net package, PAP: GREEN)
    cymru/          # ASN lookups via Team Cymru DNS (PAP: AMBER)
    crtsh/          # Certificate transparency via crt.sh (PAP: AMBER)
    threatminer/    # Threat intel via ThreatMiner API (PAP: AMBER)
    pgp/            # PGP key search via keys.openpgp.org (PAP: AMBER)
    quad9/          # Quad9 threat-intelligence blocked check via DoH (PAP: AMBER)
    detect/         # Active provider detection via DNS lookups (PAP: GREEN)
    apex/           # Aggregate DNS recon via Quad9 DoH (PAP: AMBER)
    identify/       # Offline provider detection from known record values (PAP: RED)
  output/           # Text (tablewriter), JSON, text formatters + defang
  testutil/         # Shared test helpers (mock resolver, nop logger)
```

---

## Responsible Use

trident is designed for use in **authorised environments only** — internal security assessments,
red team engagements you have permission to conduct, and OSINT research on infrastructure you
own or have been explicitly authorised to investigate.

**Malicious use is strictly prohibited.** Do not use trident to query systems or services
without authorisation. Misuse may violate computer fraud laws and the terms of service of the
queried APIs.

By default trident identifies itself honestly with a `trident/<version>` HTTP User-Agent so that
server operators can recognise and control its traffic.

---

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding standards, and the pull request process.

## Security

To report a vulnerability, see [SECURITY.md](SECURITY.md).

## Code of Conduct

This project follows the Contributor Covenant v3.0. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## License

[GPL-3.0](LICENSE.md)
