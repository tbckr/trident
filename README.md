# trident

[![CI](https://github.com/tbckr/trident/actions/workflows/ci.yml/badge.svg)](https://github.com/tbckr/trident/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/tbckr/trident)](https://github.com/tbckr/trident/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/tbckr/trident)](https://github.com/tbckr/trident/blob/main/go.mod)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/tbckr/trident/badge)](https://securityscorecards.dev/viewer/?uri=github.com/tbckr/trident)
[![CodeQL](https://github.com/tbckr/trident/actions/workflows/codeql.yml/badge.svg)](https://github.com/tbckr/trident/actions/workflows/codeql.yml)

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

**Nix** — run without installing or add to your system profile:

```bash
# Run directly
nix run github:tbckr/trident -- dns example.com

# Install to profile
nix profile install github:tbckr/trident
```

**Build from source:**

```bash
git clone https://github.com/tbckr/trident
cd trident
go build -o trident ./cmd/trident
```

---

## Verify Release Artifacts

> **Note:** Starting with **v0.10.0**, releases use [GitHub Artifact Attestation](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds) for provenance. Previous releases (v0.9.x) used `cosign attest-blob`; releases before v0.8.0 used `cosign sign-blob`.

Every release is attested via GitHub Artifact Attestation using [`actions/attest-build-provenance`](https://github.com/actions/attest-build-provenance). GitHub signs a provenance statement for every artifact listed in `checksums.txt`, proving it was built by the official release workflow. The attestation includes the artifact's SHA-256 digest, so a single `gh attestation verify` call proves both provenance and integrity.

### Manual verification

```bash
ARCHIVE=trident_Linux_x86_64.tar.gz

# Download the archive from the releases page, then verify attestation
gh attestation verify "$ARCHIVE" --repo tbckr/trident
```

### Script

`scripts/verify-release.sh` automates the step above:

```bash
# Download the archive from the releases page first, then:
./scripts/verify-release.sh trident_Linux_x86_64.tar.gz
```

The script runs `gh attestation verify` and exits non-zero on failure. It requires the [GitHub CLI](https://cli.github.com/) (2.49+).

### Checksum-only (without gh CLI)

If you do not have the GitHub CLI installed, you can still verify the archive hash against `checksums.txt` after downloading it from the releases page:

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
- **Proxy support** — HTTP, HTTPS, and SOCKS5 (`socks5h://`) proxies via `--proxy` or the standard `HTTPS_PROXY`/`HTTP_PROXY`/`ALL_PROXY`/`NO_PROXY` variables, resolved once with curl-like rules; see [Proxy](#proxy)
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
proxy: socks5h://127.0.0.1:9050
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

### Proxy

trident resolves its proxy once per run and uses that one answer for every network path: https
requests, http requests, and DNS lookups. `trident config show` prints the result (see
[`config`](#config--configuration-management)).

**Which proxy applies.** `--proxy` (or `TRIDENT_PROXY`, or `proxy:` in the config file) applies to
all three paths, and every proxy environment variable, `NO_PROXY` included, is then ignored. This
differs from curl, which still applies `NO_PROXY` to `--proxy`. Without it, the environment
decides; each variable may be spelled in lowercase or uppercase:

| Path | Proxy used |
|------|------------|
| https requests | `https_proxy` / `HTTPS_PROXY`, else `all_proxy` / `ALL_PROXY` |
| http requests | `http_proxy` / `HTTP_PROXY`, else `all_proxy` / `ALL_PROXY` |
| DNS lookups | the https value, so DNS and HTTPS leave through the same exit |

Every trident service talks HTTPS, so `HTTP_PROXY` alone is not enough (see Errors below). With
only an https proxy set, `http://` targets have no proxy, and trident refuses to connect to them
directly, because a proxy is configured for https; `config show` reads
`proxy_http=refused (no http proxy; https is proxied)`. A host that `NO_PROXY` exempts is still
reached directly, with the exemption warning. This matters only for `http://` redirects and a
custom `detect_patterns.url`. When a server runs trident as a CGI
script (`REQUEST_METHOD` is set), uppercase `HTTP_PROXY` is ignored, because the web server fills it
from the request's `Proxy:` header; lowercase `http_proxy` is kept.

**Accepted values.** `http://`, `https://`, `socks5://` and `socks5h://`, with the scheme in any
case. A value without `://` is read as `http://` plus the value (`proxy.corp:3128`). A user name
and password (`user:pass@host`) are allowed; for a SOCKS proxy they become SOCKS username and
password authentication. With `socks5://` as well as `socks5h://`, trident sends target host names
to the proxy and never resolves them locally; `socks5h://` is the spelling that says so. Not
accepted: `socks4://` and `socks4a://` (SOCKS4 makes the client resolve names itself), `socks://`
(ambiguous; use `socks5h://`), whitespace or control characters, a missing host, a port outside
1-65535, and a path (other than a trailing `/`), query or fragment. trident never trims or repairs
a value, so a leading space is an error and not a silent direct connection. The proxy's own host
name is resolved locally on every connection (inherent to Go and the HTTP library): prefer an IP
literal, as in `socks5h://127.0.0.1:9050`.

**`NO_PROXY`** follows curl's rules, and only applies to proxies taken from the environment:

- Entries are separated by commas; spaces and case do not matter.
- `example.com`, `.example.com` and `*.example.com` all match the host itself and every
  subdomain, but not `badexample.com`. (`*.example.com` is trident's extension; curl has no `*.`
  wildcard.)
- IP addresses and CIDR ranges (IPv4 and IPv6) match only targets given as IP literals; no name
  is looked up for matching. An entry with a port (`host:port`, `[::1]:8080`) matches only that
  port. Internationalised names are compared in their ASCII (IDNA) form.
- There is no implicit exemption for `localhost` or loopback addresses. If a local service (for
  example a patterns server) must be reached directly, add it: `NO_PROXY=localhost,127.0.0.1`.
- `NO_PROXY` does not affect DNS lookups, except for `*`.
- A `*` among other entries (`localhost,*`) is an error: Go would bypass the proxy for everything
  and curl for nothing. Use `NO_PROXY=*` alone.
- When `NO_PROXY` exempts a host that trident is about to contact while a proxy is configured,
  trident connects directly and logs a warning once per host: `NO_PROXY exempts this host from
  the proxy; connecting directly`, with the host name attached.
- **`NO_PROXY=*`** turns the proxy off for https, http and DNS alike, with no warning and no
  validation of the proxy variables. This is the way to say "connect directly, on purpose" while
  proxy variables are exported.

**Errors.** When trident cannot honour a proxy setting it stops instead of connecting around the
proxy. Only the commands that use the network stop; `identify`, `config`, `version` and shell
completion keep working. Two situations only warn: a `NO_PROXY` exemption (above) and DNS behind
an HTTP(S) proxy (below). Errors read `invalid proxy configuration: <source>: <reason>`, where the
source is `proxy` (the flag, config file or `TRIDENT_PROXY`) or the exact variable name; the
lowercase/uppercase conflict and the http-only error name their variables in the sentence instead.
They never contain the proxy value, and no output ever shows a user name or password.

| Situation | What to do |
|-----------|------------|
| Only `http_proxy` / `HTTP_PROXY` is set. It covers `http://` URLs only, so every https request would bypass it: `HTTP_PROXY applies only to http:// URLs, but trident's services use https:// and would bypass the proxy; ...` | Set `HTTPS_PROXY` (or `ALL_PROXY`) to the same value, use `--proxy socks5h://...`, or set `NO_PROXY='*'` to connect directly. |
| A lowercase variable and its uppercase twin are both set and differ: `https_proxy and HTTPS_PROXY are both set and differ; unset one` | Unset one of the two. An empty variable counts as unset. |
| A selected value is not usable (`socks4://`, `socks://`, a leading space, a path, ...): for example `HTTPS_PROXY: socks4:// is not supported; use socks5h://` | Fix the value. Only values trident actually uses are checked: a broken `ALL_PROXY` next to valid `HTTPS_PROXY` and `HTTP_PROXY` is ignored. |
| `NO_PROXY` has a `*` among other entries (checked only when a proxy applies) | Use `NO_PROXY=*` alone, or list the hosts. |
| A connection would reach a proxied host directly, for example an `http://` URL while only an https proxy is set: `refusing direct connection while a proxy is configured` | Set `http_proxy` or `ALL_PROXY` as well, or exempt the host with `NO_PROXY`. |

**DNS.** `dns`, `detect` and `cymru` (and `apex`, see below) look names up as follows:

- **SOCKS5 proxy** (`socks5://` or `socks5h://` is the https value): every query goes over TCP
  through the proxy to Quad9's unfiltered resolver at `9.9.9.10:53`, never to the nameserver in
  `/etc/resolv.conf`: that is often a local stub such as `127.0.0.53`, which a remote proxy would
  read as its own loopback, and Tor refuses it. Names are sent fully qualified, so the search
  domains from `resolv.conf` never leave through the proxy. The nameserver is fixed for now, and
  your proxy must allow TCP connections to `9.9.9.10:53`. Quad9 on this address does not validate
  DNSSEC.
- **HTTP or HTTPS proxy:** a proxy like that cannot carry DNS, so the system resolver is used,
  outside the proxy, and trident warns: `DNS lookups cannot go through the configured proxy and use
  the system resolver; use a socks5h:// proxy (apex already resolves through the proxy via DoH)`,
  with the redacted proxy URL attached. The warning appears only in commands that look names up
  this way, not in `crtsh` or `pgp`. Use a SOCKS5 proxy, or `apex`, which asks Quad9 over HTTPS and
  so goes through the proxy.
- **No proxy, or `NO_PROXY=*`:** the system resolver, with no warning.

A VPN such as Mullvad needs no proxy setting: trident's traffic, DNS lookups included, already
leaves through the tunnel. Turn on the VPN client's lockdown mode (Mullvad: "Lockdown mode") so
that nothing leaves outside the tunnel while it is down.

**How requests behave behind a proxy.** The HTTP-based services speak HTTP/1.1 only, and trident
refuses any connection that would reach a proxied host directly instead of through the proxy
(the request fails with "refusing direct connection while a proxy is configured"). This works
around HTTP/2 code paths in the HTTP library that ignore the proxy
([imroc/req#541](https://github.com/imroc/req/pull/541)).

`quad9` and `apex` are the exception: Quad9 no longer accepts DNS-over-HTTPS over HTTP/1.1
(it answers `505`), so these two commands speak HTTP/2 through the proxy, on a separate
transport that never connects around it. They work through `http://`, `https://`, `socks5://` and
`socks5h://` proxies. A TLS-intercepting proxy that only speaks HTTP/1.1 cannot carry
DNS-over-HTTPS: Quad9 answers `505` and the command fails with an error. An `https://` proxy that
itself negotiates HTTP/2 (Envoy and Caddy can, Squid does not) breaks the connection to it for
these two commands, which then fail with an error as well; use an `http://` or `socks5h://` proxy
instead. Behind a proxy that applies to `dns.quad9.net` (that is, `--proxy`, `HTTPS_PROXY` or
`ALL_PROXY`, and not exempted by `NO_PROXY`), `apex` also sends its Team Cymru ASN lookups through
DNS-over-HTTPS instead of plain DNS from the local resolver. If one of those lookups would bypass
the proxy, `apex` stops with an error.

**If you used proxies with an earlier version,** these behaviours changed:

- `ALL_PROXY` alone now proxies all HTTP services and the DNS-over-HTTPS queries (before, it
  affected DNS lookups only).
- With only `http_proxy` / `HTTP_PROXY` set, network commands now stop with an error (before,
  everything went direct without a word).
- A lowercase/uppercase mismatch, a broken value, and `socks://` now stop the network commands.
- `--proxy` accepts `socks5h://`, and schemes in any case. DNS through a SOCKS proxy works with
  `socks5h://`, uppercase schemes, a trailing `/` and credentials (before, such values sent DNS
  direct or made every lookup fail), and goes to `9.9.9.10` instead of the system's nameserver.
- `NO_PROXY` follows curl's rules, with no implicit `localhost` exemption, and exemptions of the
  hosts trident contacts are warned about.
- The DNS warning appears only in the commands that use the system resolver (before, it appeared
  in every command run with a `--proxy` or `ALL_PROXY` other than a lowercase `socks5://` one,
  `crtsh` and `pgp` included).
- `config show` lists the proxy per path and never prints credentials.

---

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | platform config dir | Config file path |
| `--verbose`, `-v` | `false` | Enable debug logging |
| `--output`, `-o` | `table` | Output format: `table`, `json`, `text` |
| `--concurrency`, `-c` | `10` | Worker pool size for bulk input |
| `--proxy` | — | Proxy URL (`http://`, `https://`, `socks5://`, `socks5h://`; scheme in any case). Overrides the proxy environment variables, `NO_PROXY` included |
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
IP address. Makes direct queries to the configured DNS resolver (PAP: GREEN). Behind a SOCKS5
proxy the queries go through the proxy to Quad9 at `9.9.9.10` instead; see [Proxy](#proxy).

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

**Proxy rows.** Besides `proxy` (trident's own setting: `--proxy`, the config file or
`TRIDENT_PROXY`; empty when only the environment is set), `config show` has four read-only rows
that report the proxy trident actually uses per path, and where it comes from. See
[Proxy](#proxy) for the rules.

| Row | Value |
|-----|-------|
| `proxy_https` | The proxy for https requests and its source (a variable name, or `proxy` for trident's own setting), or `direct`, or `direct (NO_PROXY=*)` |
| `proxy_http` | The same for http requests; `refused (no http proxy; https is proxied)` when only https has a proxy (trident then refuses `http://` URLs instead of connecting directly) |
| `proxy_dns` | Over SOCKS: `<proxy> (<source>), nameserver 9.9.9.10:53`. Behind an HTTP(S) proxy: `system resolver, not proxied (<source> is not a SOCKS5 proxy)`. Otherwise `system resolver`, or `system resolver (NO_PROXY=*)` |
| `proxy_no_proxy` | The `NO_PROXY` value in effect (empty when `proxy` is set) |

```console
$ HTTPS_PROXY=http://proxy.corp:3128 HTTP_PROXY=http://proxy.corp:3128 NO_PROXY=.corp,localhost \
    trident config show -o text | grep '^proxy'
proxy=
proxy_dns=system resolver, not proxied (HTTPS_PROXY is not a SOCKS5 proxy)
proxy_http=http://proxy.corp:3128 (HTTP_PROXY)
proxy_https=http://proxy.corp:3128 (HTTPS_PROXY)
proxy_no_proxy=.corp,localhost

$ trident --proxy socks5h://127.0.0.1:9050 config show -o text | grep '^proxy'
proxy=socks5h://127.0.0.1:9050
proxy_dns=socks5h://127.0.0.1:9050 (proxy), nameserver 9.9.9.10:53
proxy_http=socks5h://127.0.0.1:9050 (proxy)
proxy_https=socks5h://127.0.0.1:9050 (proxy)
proxy_no_proxy=
```

No row ever shows credentials: a proxy URL with a user name or password appears as
`scheme://***@host:port`. If a proxy setting cannot be honoured, the affected rows read
`error: <message>`, the message the network commands stop with; `config show` itself keeps
working.

**Limitations:**
- `config show` and `config get` report *effective* values — the result of merging built-in
  defaults, `TRIDENT_*` environment variables, and the config file. They do not show what is
  literally written in the file.
- `config get` also reads the four `proxy_*` rows; `config set` rejects them, because they only
  report. `config set proxy <value>` checks the value like `--proxy` does (an invalid value is
  refused without being echoed back).
- `config set` writes to the file but takes effect on the **next invocation**; the current
  process already loaded config at startup.
- The `aliases` section is not managed by `config set` — use the `alias` subcommand instead.
- Only known configuration keys are accepted (`output`, `pap_limit`, `proxy`, `user_agent`,
  `concurrency`, `verbose`, `defang`, `no_defang`, `detect_patterns.url`, `detect_patterns.file`).
  `proxy` takes `http://`, `https://`, `socks5://` or `socks5h://` (scheme in any case).

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

### Nix Dev Shell

If you use [Nix](https://nixos.org/), `nix develop` provides Go, golangci-lint, goreleaser, just, semgrep, betterleaks, and svu. Entering the shell also enables a local pre-commit hook (`.githooks/pre-commit`) that scans staged changes for secrets via betterleaks:

```bash
nix develop
```

For non-Nix users, enable the same hook manually (one-time):

```bash
git config core.hooksPath .githooks
```

### justfile

A [justfile](https://github.com/casey/just) provides convenient targets for common tasks:

```bash
# Build & Test
just build              # Build all packages
just test               # Run all tests with coverage
just test-pkg ./internal/services/dns/...  # Test a specific package
just test-race          # Run all tests with race detector
just fuzz ./internal/output/...            # Run fuzz tests for a package
just coverage           # Check service coverage meets 80% threshold

# Code Quality
just fmt                # Format all Go files with gofmt
just lint               # Run golangci-lint
just tidy               # Tidy and verify modules
just tidy-check         # Verify modules are tidy (fails if dirty)
just vuln               # Run govulncheck
just license-check      # Check dependency licenses against allowlist
just semgrep            # Pattern-based SAST (custom rules + p/golang + p/gosec)
just leaks              # Secret scanning via betterleaks

# CI
just ci                 # Run all CI checks locally

# Nix
just flake-build        # Build the Nix package locally
just flake-check        # Run Nix flake check
just flake-update       # Update Nix flake inputs

# Release
just release            # Tag next version with svu and push
just goreleaser-check   # Validate .goreleaser.yaml config
just verify-release trident_Linux_x86_64.tar.gz  # Verify release artifact

# Maintenance
just outdated           # List newer dependencies, tools and images to review
just upgrade            # Upgrade direct dependencies
just harden-repo        # Apply repository hardening settings
```

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
cmd/docgen/         # Man pages + shell completions generator (cobra/doc)
internal/
  cli/              # Cobra command tree, global flags, output wiring
  config/           # Viper config loading and flag registration
  httpclient/       # req.Client factories (proxy, UA rotation, debug tracing; HTTP/2 DoH client)
  proxyconf/        # The one proxy resolution: --proxy/config + env, curl-like NO_PROXY, validation, redaction
  input/            # Line reader from io.Reader for stdin path
  pap/              # PAP level constants and enforcement
  doh/              # DNS-over-HTTPS client (Quad9 RFC 8484, shared by apex + quad9)
  ratelimit/        # Token-bucket rate limiter with ±20% jitter
  resolver/         # DNS resolver factory: system resolver, or SOCKS5 to a pinned nameserver
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
  appdir/           # OS config-dir helpers: ConfigDir(), EnsureFile()
  apperr/           # Shared error sentinels (leaf; no internal imports)
  detect/           # Provider detection: CDN/Email/DNS/TXT (pure, no I/O); patterns.yaml embedded
  output/           # Text (tablewriter), JSON, text formatters + defang
  testutil/         # Shared test helpers (mock resolver, nop logger)
  version/          # Build version info (ldflags + BuildInfo fallback)
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

trident follows a defense-in-depth approach to supply chain security:

- **Static analysis** — CodeQL SAST on every push/PR and weekly scans
- **Pattern-based SAST** — Semgrep with custom project rules + `p/golang` + `p/gosec` on every push/PR and weekly scans
- **Vulnerability scanning** — govulncheck on every push/PR and daily scheduled scans
- **OpenSSF Scorecard** — weekly independent assessment of security posture
- **Release provenance** — GitHub Artifact Attestation (SLSA) for every release artifact
- **SBOM** — CycloneDX software bill of materials included with every release
- **VEX** — OpenVEX vulnerability assessment document included with every release (govulncheck reachability analysis)
- **Hardened CI** — SHA-pinned GitHub Actions, least-privilege permissions, sandboxed steps
- **Repository protection** — GitHub Rulesets for branch and tag integrity

For full details, see [docs/supply-chain-security.md](docs/supply-chain-security.md).

To report a vulnerability, see [SECURITY.md](SECURITY.md).

## Code of Conduct

This project follows the Contributor Covenant v3.0. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## License

[GPL-3.0](LICENSE.md)
