# Trident

**Trident** is a fast, keyless OSINT CLI for network reconnaissance. It is a Go port and evolution of the Python [Harpoon](https://github.com/Te-k/harpoon) tool — single binary, no API keys required, built for analysts and security researchers.

## Quickstart

**Prerequisites:** Go 1.26+ (`go version`)

```bash
go install github.com/tbckr/trident/cmd/trident@latest
```

```bash
# DNS records for a domain
$ trident dns example.com

# Subdomains from certificate transparency logs
$ trident crtsh example.com

# ASN info for an IP or ASN number
$ trident asn 8.8.8.8
```

## Features

- **No API keys** — all current services are keyless
- **Bulk input** — pipe targets via stdin or pass multiple arguments
- **Three output formats** — `text` (tables), `json`, and `plain` (one result per line for piping)
- **Proxy support** — HTTP, HTTPS, and SOCKS5 proxies
- **PAP system** — Permissible Actions Protocol (RED/AMBER/GREEN/WHITE) to prevent accidental active interaction
- **Auto-defanging** — URLs and IPs are defanged at strict PAP levels
- **Rate limiting** — per-service token-bucket rate limiter with jitter to avoid detectable request patterns
- **Concurrent processing** — configurable worker pool for bulk lookups
- **Cross-platform** — Linux, macOS, Windows

## Installation

### Pre-built binaries

Download the latest release from the [GitHub releases page](https://github.com/tbckr/trident/releases). Archives are available for Linux, macOS, and Windows (amd64 and arm64).

Linux users can install via package managers using the `.deb`, `.rpm`, or `.apk` packages included in each release.

### Go install

```bash
go install github.com/tbckr/trident/cmd/trident@latest
```

### Build from source

```bash
git clone https://github.com/tbckr/trident
cd trident
go build -o trident ./cmd/trident
```

## Usage

### Common Workflows

```bash
# DNS records for a domain or reverse PTR lookup for an IP
trident dns example.com
trident dns 8.8.8.8

# ASN info for an IP or ASN number (IPv4 and IPv6 supported)
trident asn 8.8.8.8
trident asn AS15169
trident asn 2001:4860:4860::8888

# Subdomains from certificate transparency logs
trident crtsh example.com

# Threat intelligence (domain, IP, or file hash)
trident threatminer example.com
trident threatminer 198.51.100.1
trident threatminer d41d8cd98f00b204e9800998ecf8427e

# PGP key search by email or name
trident pgp alice@example.com
trident pgp "Alice Smith"
```

### Bulk Input

Any command accepts multiple targets as arguments or from stdin (one per line):

```bash
# Multiple arguments
trident dns example.com google.com cloudflare.com

# From a file via stdin
cat targets.txt | trident crtsh

# From another command
cat /etc/hosts | awk '{print $1}' | trident asn

# Control concurrency for large lists
cat ips.txt | trident asn --concurrency=20
```

### Output Formats

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

## Commands

### `dns` — DNS Lookups (PAP: GREEN)

Resolves A, AAAA, MX, NS, and TXT records for a domain, or performs a reverse PTR lookup for an IP
address. Makes direct queries to the configured DNS resolver.

```bash
trident dns example.com
trident dns 8.8.8.8
trident dns 2001:4860:4860::8888
```

### `asn` — ASN Lookup (PAP: AMBER)

Looks up ASN information for an IP address or ASN number via the Team Cymru DNS service. Supports
both IPv4 and IPv6.

```bash
trident asn 8.8.8.8
trident asn AS15169
trident asn 2001:4860:4860::8888
```

### `crtsh` — Certificate Transparency (PAP: AMBER)

Searches [crt.sh](https://crt.sh) certificate transparency logs for subdomains of a domain.

```bash
trident crtsh example.com
```

### `threatminer` — Threat Intelligence (PAP: AMBER)

Queries the [ThreatMiner](https://www.threatminer.org) API for contextual threat intelligence.
Automatically detects whether input is a domain, IP address, or file hash. Rate-limited to 1
request/second with jitter to avoid triggering ThreatMiner's rate limits.

```bash
trident threatminer example.com
trident threatminer 198.51.100.1
trident threatminer d41d8cd98f00b204e9800998ecf8427e
```

### `pgp` — PGP Key Search (PAP: AMBER)

Searches [keys.openpgp.org](https://keys.openpgp.org) for PGP keys by email address or name using
the HKP protocol.

```bash
trident pgp alice@example.com
trident pgp "Alice Smith"
```

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | platform config dir | Config file path |
| `--verbose`, `-v` | `false` | Enable debug logging |
| `--output`, `-o` | `text` | Output format: `text`, `json`, `plain` |
| `--concurrency`, `-c` | `10` | Worker pool size for bulk input |
| `--proxy` | — | Proxy URL (`http://`, `https://`, `socks5://`) |
| `--user-agent` | rotating browser UAs | Override HTTP User-Agent |
| `--pap-limit` | `white` | PAP limit: `red`, `amber`, `green`, `white` |
| `--defang` | `false` | Force output defanging |
| `--no-defang` | `false` | Disable output defanging |

## PAP System

Trident implements the [Permissible Actions Protocol (PAP)](https://www.misp-project.org/taxonomies.html#_pap)
to prevent accidental active interaction with targets:

| Level | Meaning | Services |
|-------|---------|----------|
| `red` | Non-detectable, offline/local only | — |
| `amber` | Limited 3rd-party APIs, no direct target contact | ASN, crt.sh, ThreatMiner, PGP |
| `green` | Direct target interaction permitted | DNS |
| `white` | Unrestricted (default) | all |

Set a limit to block services above that level:

```bash
# Only run services that use 3rd-party APIs (no direct target contact)
trident --pap-limit=amber crtsh example.com

# Block all active interaction
trident --pap-limit=red asn 8.8.8.8  # error: service level AMBER exceeds limit RED
```

Defanging is automatically applied at AMBER and below unless `--no-defang` is passed.

## Configuration

The config file is created automatically at first run. The default location is platform-specific:

- **Linux:** `$XDG_CONFIG_HOME/trident/config.yaml` (typically `~/.config/trident/config.yaml`)
- **macOS:** `~/Library/Application Support/trident/config.yaml`
- **Windows:** `%AppData%\trident\config.yaml`

File permissions are set to `0600`. All flags can be persisted in the config file:

```yaml
output: json
pap_limit: amber
concurrency: 20
proxy: socks5://127.0.0.1:9050
```

Environment variables override config file values using the `TRIDENT_` prefix:

| Variable | Corresponding flag |
|----------|--------------------|
| `TRIDENT_OUTPUT` | `--output` |
| `TRIDENT_PAP_LIMIT` | `--pap-limit` |
| `TRIDENT_PROXY` | `--proxy` |
| `TRIDENT_USER_AGENT` | `--user-agent` |
| `TRIDENT_CONCURRENCY` | `--concurrency` |
| `TRIDENT_VERBOSE` | `--verbose` |
| `TRIDENT_DEFANG` | `--defang` |
| `TRIDENT_NO_DEFANG` | `--no-defang` |

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

## Contributing

Contributions are welcome. Please open an issue before implementing a significant change to discuss
the approach.

## License

[GPL-3.0](LICENSE)
