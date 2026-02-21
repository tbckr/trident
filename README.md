# Trident

**Trident** is a fast, keyless OSINT CLI for network reconnaissance. It is a Go port and evolution of the Python [Harpoon](https://github.com/Te-k/harpoon) tool — single binary, no API keys required, built for analysts and security researchers.

Gather DNS records, ASN information, certificate transparency subdomains, threat intelligence, and PGP keys in one consistent interface. Pipe results between tools, process bulk inputs from stdin, and control OpSec posture with the built-in PAP system.

## Features

- **No API keys** — all current services are keyless
- **Bulk input** — pipe targets via stdin or pass multiple args
- **Three output formats** — `text` (tables), `json`, and `plain` (one result per line for piping)
- **Proxy support** — HTTP, HTTPS, and SOCKS5 proxies
- **PAP system** — Permissible Actions Protocol (RED/AMBER/GREEN/WHITE) to prevent accidental active interaction
- **Auto-defanging** — URLs and IPs are defanged at strict PAP levels
- **Concurrent processing** — configurable worker pool for bulk lookups
- **Cross-platform** — Linux, macOS, Windows

## Installation

```bash
go install github.com/tbckr/trident/cmd/trident@latest
```

Or build from source:

```bash
git clone https://github.com/tbckr/trident
cd trident
go build -o trident ./cmd/trident
```

**Requirements:** Go 1.26+

## Quick Start

```bash
# DNS records for a domain
trident dns example.com

# ASN info for an IP or ASN number
trident asn 8.8.8.8
trident asn AS15169

# Subdomains from certificate transparency logs
trident crtsh example.com

# Threat intelligence (domain, IP, or file hash)
trident threatminer example.com
trident threatminer 1.2.3.4
trident threatminer d41d8cd98f00b204e9800998ecf8427e

# PGP key search
trident pgp alice@example.com

# Bulk input via stdin
echo -e "8.8.8.8\n1.1.1.1" | trident asn
cat domains.txt | trident dns

# JSON output for scripting
trident crtsh example.com -o json | jq '.subdomains[]'

# Plain output for piping
trident crtsh example.com -o plain | sort -u
```

## Commands

### `dns` — DNS Lookups (PAP: GREEN)

Resolves A, AAAA, MX, NS, and TXT records for a domain, or performs a reverse PTR lookup for an IP address.

```bash
trident dns example.com
trident dns 8.8.8.8
trident dns 2001:4860:4860::8888
```

### `asn` — ASN Lookup (PAP: AMBER)

Looks up ASN information for an IP address or ASN number via the Team Cymru DNS service. Supports both IPv4 and IPv6.

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

Queries the [ThreatMiner](https://www.threatminer.org) API for contextual threat intelligence. Automatically detects whether input is a domain, IP address, or file hash.

```bash
trident threatminer example.com
trident threatminer 198.51.100.1
trident threatminer d41d8cd98f00b204e9800998ecf8427e
```

### `pgp` — PGP Key Search (PAP: AMBER)

Searches [keys.openpgp.org](https://keys.openpgp.org) for PGP keys by email address or name using the HKP protocol.

```bash
trident pgp alice@example.com
trident pgp "Alice Smith"
```

## Output Formats

**Text (default)** — formatted ASCII tables, ideal for human reading:

```bash
trident dns example.com
trident asn AS15169 -o text
```

**JSON** — structured output for scripting and integration:

```bash
trident dns example.com -o json
trident crtsh example.com -o json | jq '.subdomains | length'
```

**Plain** — one result per line, ideal for piping into other tools:

```bash
trident crtsh example.com -o plain | sort -u > subdomains.txt
trident dns example.com -o plain | grep "^A "
```

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `~/.config/trident/config.yaml` | Config file path |
| `--verbose`, `-v` | `false` | Enable debug logging |
| `--output`, `-o` | `text` | Output format: `text`, `json`, `plain` |
| `--concurrency`, `-c` | `10` | Worker pool size for bulk input |
| `--proxy` | — | Proxy URL (`http://`, `https://`, `socks5://`) |
| `--user-agent` | rotating browser UAs | Override HTTP User-Agent |
| `--pap` | `white` | PAP limit: `red`, `amber`, `green`, `white` |
| `--defang` | `false` | Force output defanging |
| `--no-defang` | `false` | Disable output defanging |

## PAP System

Trident implements the [Permissible Actions Protocol (PAP)](https://www.misp-project.org/taxonomies.html#_pap) to prevent accidental active interaction with targets:

| Level | Color | Meaning | Services |
|-------|-------|---------|---------|
| `red` | RED | Non-detectable, offline/local only | — |
| `amber` | AMBER | Limited 3rd-party APIs, no direct target contact | ASN, crt.sh, ThreatMiner, PGP |
| `green` | GREEN | Direct target interaction permitted | DNS |
| `white` | WHITE | Unrestricted (default) | all |

Set a limit to block services above that level:

```bash
# Only run services that don't touch the target at all
trident --pap=amber crtsh example.com

# Block all active interaction
trident --pap=red asn 8.8.8.8  # error: service level AMBER exceeds limit RED
```

Defanging is automatically applied at AMBER and below unless `--no-defang` is passed.

## Bulk Input

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

## Configuration

The config file is created automatically at first run:

- **Linux/macOS:** `~/.config/trident/config.yaml`
- **Windows:** `%APPDATA%\trident\config.yaml`

File permissions are set to `0600`. All flags can be persisted in the config file:

```yaml
output: json
pap: amber
concurrency: 20
proxy: socks5://127.0.0.1:9050
```

Environment variables override config file values using the `TRIDENT_` prefix:

```bash
TRIDENT_OUTPUT=json trident dns example.com
TRIDENT_PAP=amber trident crtsh example.com
TRIDENT_PROXY=socks5://127.0.0.1:9050 trident asn 8.8.8.8
```

## Development

### Requirements

- Go 1.26+
- [golangci-lint](https://golangci-lint.run/) v2

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
  pap/              # PAP level constants and enforcement
  worker/           # Bounded goroutine pool + stdin reader
  services/         # One package per OSINT service
    dns/            # DNS record lookups (net package, PAP: GREEN)
    asn/            # ASN lookups via Team Cymru DNS (PAP: AMBER)
    crtsh/          # Certificate transparency via crt.sh (PAP: AMBER)
    threatminer/    # Threat intel via ThreatMiner API (PAP: AMBER)
    pgp/            # PGP key search via keys.openpgp.org (PAP: AMBER)
  output/           # Text (tablewriter), JSON, plain formatters + defang
  validate/         # Shared input validators
  testutil/         # Shared test helpers (mock resolver, nop logger)
```

## Contributing

Contributions are welcome. Please open an issue before implementing a significant change to discuss the approach.

## License

See [LICENSE](LICENSE).
