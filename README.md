# trident

[![CI](https://github.com/tbckr/trident/actions/workflows/ci.yml/badge.svg)](https://github.com/tbckr/trident/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/tbckr/trident)](https://github.com/tbckr/trident/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/tbckr/trident)](https://github.com/tbckr/trident/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/tbckr/trident)](https://goreportcard.com/report/github.com/tbckr/trident)
[![License: GPL-3.0](https://img.shields.io/github/license/tbckr/trident)](LICENSE)

**Fast, keyless OSINT in a single binary.** DNS lookups, ASN info, certificate transparency, threat intelligence, and PGP key search — no API keys, no registration, no configuration required.

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

---

## Installation

**The fastest way** — requires Go 1.26+:

```bash
go install github.com/tbckr/trident/cmd/trident@latest
```

**Pre-built binaries** — download for Linux, macOS, or Windows (amd64/arm64) from the [releases page](https://github.com/tbckr/trident/releases). Linux packages (`.deb`, `.rpm`, `.apk`) are included.

**Build from source:**

```bash
git clone https://github.com/tbckr/trident
cd trident
go build -o trident ./cmd/trident
```

---

## Quickstart

```bash
# DNS records — forward lookup or reverse PTR
trident dns example.com
trident dns 8.8.8.8

# ASN info — IP address or ASN number (IPv4 and IPv6)
trident asn 8.8.8.8
trident asn AS15169

# Subdomains from certificate transparency logs
trident crtsh example.com

# Threat intelligence — domain, IP, or file hash
trident threatminer example.com
trident threatminer d41d8cd98f00b204e9800998ecf8427e

# PGP key search — by email, name, or fingerprint
trident pgp alice@example.com
trident pgp 0xDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF
```

---

## Features

- **No API keys** — all current services are keyless; install and run immediately
- **Bulk input** — pipe a target list via stdin or pass multiple arguments
- **Three output formats** — `text` (tables), `json`, and `plain` (one result per line for piping)
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
| `asn` | ASN info for IPs and ASN numbers (IPv4 + IPv6) | AMBER | Team Cymru DNS |
| `crtsh` | Subdomain enumeration via certificate transparency | AMBER | [crt.sh](https://crt.sh) |
| `threatminer` | Threat intel for domains, IPs, and file hashes | AMBER | [ThreatMiner](https://www.threatminer.org) |
| `pgp` | PGP key search by email, name, or fingerprint | AMBER | [keys.openpgp.org](https://keys.openpgp.org) |

---

## Output Formats

**Text (default)** — formatted ASCII tables for human reading:

```bash
trident dns example.com
trident asn AS15169 -o text
```

**JSON** — structured output for scripting and integration:

```bash
trident dns example.com -o json
trident crtsh example.com -o json | jq '.subdomains | length'
```

**Plain** — one result per line, ideal for piping:

```bash
trident crtsh example.com -o plain | sort -u > subdomains.txt
trident dns example.com -o plain | grep "^A "
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
cat /etc/hosts | awk '{print $1}' | trident asn

# Control concurrency for large lists
cat ips.txt | trident asn --concurrency=20
```

---

## PAP System

trident implements the [Permissible Actions Protocol (PAP)](https://www.misp-project.org/taxonomies.html#_pap)
to prevent accidental active interaction with targets:

| Level | Meaning | Permitted Services |
|-------|---------|-------------------|
| `red` | Offline/local only — non-detectable | none |
| `amber` | Limited 3rd-party APIs — no direct target contact | ASN, crt.sh, ThreatMiner, PGP |
| `green` | Direct target interaction permitted | DNS + all AMBER |
| `white` | Unrestricted **(default)** | all |

Set `--pap-limit` to block services above that level:

```bash
# Only use 3rd-party APIs (no direct DNS queries to the target)
trident --pap-limit=amber crtsh example.com

# This will error — AMBER exceeds RED limit
trident --pap-limit=red asn 8.8.8.8
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
edit directly. The config file supports all global flags plus the `alias` block:

```yaml
output: json
pap_limit: amber
concurrency: 20
proxy: socks5://127.0.0.1:9050
alias:
  ct: crtsh
  myasn: "asn --pap-limit=amber"
```

> **Note:** The `alias` block is config-file only — it has no corresponding flag or environment
> variable. Use `trident alias set` / `trident alias delete` to manage aliases, or edit the
> file directly.

Environment variables override config file values using the `TRIDENT_` prefix:

| Variable | Corresponding Flag |
|----------|--------------------|
| `TRIDENT_OUTPUT` | `--output` |
| `TRIDENT_PAP_LIMIT` | `--pap-limit` |
| `TRIDENT_PROXY` | `--proxy` |
| `TRIDENT_USER_AGENT` | `--user-agent` |
| `TRIDENT_CONCURRENCY` | `--concurrency` |
| `TRIDENT_VERBOSE` | `--verbose` |
| `TRIDENT_DEFANG` | `--defang` |
| `TRIDENT_NO_DEFANG` | `--no-defang` |

When `--proxy` / `TRIDENT_PROXY` is not set, trident honours the standard `HTTP_PROXY`,
`HTTPS_PROXY`, and `NO_PROXY` environment variables automatically.

---

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | platform config dir | Config file path |
| `--verbose`, `-v` | `false` | Enable debug logging |
| `--output`, `-o` | `text` | Output format: `text`, `json`, `plain` |
| `--concurrency`, `-c` | `10` | Worker pool size for bulk input |
| `--proxy` | — | Proxy URL (`http://`, `https://`, `socks5://`) |
| `--user-agent` | `trident/<version> (+https://github.com/tbckr/trident)` | Override HTTP User-Agent |
| `--pap-limit` | `white` | PAP limit: `red`, `amber`, `green`, `white` |
| `--defang` | `false` | Force output defanging |
| `--no-defang` | `false` | Disable output defanging |

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

### `asn` — ASN Lookup

Looks up ASN information for an IP address or ASN number via the Team Cymru DNS service. Supports
both IPv4 and IPv6 (PAP: AMBER).

```bash
trident asn 8.8.8.8
trident asn AS15169
trident asn 2001:4860:4860::8888
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
  `concurrency`, `verbose`, `defang`, `no_defang`).

### `alias` — Command Aliases

Define short names that expand to longer command strings. Aliases are stored in the config file
and appear in `trident --help` under *Aliases:*.

```bash
# Create or update an alias
trident alias set ct "crtsh --pap-limit=amber"

# Use the alias — extra arguments are appended after the expansion
trident ct example.com

# List all aliases
trident alias list
trident alias list -o json

# Delete an alias
trident alias delete ct
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
- Alias names cannot shadow built-in commands (`dns`, `asn`, `crtsh`, `threatminer`, `pgp`).
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
  ratelimit/        # Token-bucket rate limiter with ±20% jitter
  resolver/         # net.Resolver factory with SOCKS5 DNS-leak prevention
  worker/           # Bounded goroutine pool for bulk input
  services/         # One package per OSINT service
    dns/            # DNS record lookups (net package, PAP: GREEN)
    asn/            # ASN lookups via Team Cymru DNS (PAP: AMBER)
    crtsh/          # Certificate transparency via crt.sh (PAP: AMBER)
    threatminer/    # Threat intel via ThreatMiner API (PAP: AMBER)
    pgp/            # PGP key search via keys.openpgp.org (PAP: AMBER)
  output/           # Text (tablewriter), JSON, plain formatters + defang
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

trident identifies itself honestly with a `trident/<version>` HTTP User-Agent so that server
operators can recognise and control its traffic.

---

## Contributing

Contributions are welcome. Please open an issue before implementing a significant change to discuss
the approach.

## License

[GPL-3.0](LICENSE)
