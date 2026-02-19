# Trident

Trident is a fast, keyless OSINT CLI for DNS, ASN, and certificate transparency lookups. It is a Go port of the Python [Harpoon](https://github.com/Te-k/harpoon) tool.

**No API keys required** for Phase 1 services.

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

## Usage

```
trident [flags] <command> <input>
```

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `~/.config/trident/config.yaml` | Config file path |
| `--verbose`, `-v` | `false` | Enable debug logging |
| `--output`, `-o` | `text` | Output format: `text` or `json` |

### Commands

#### `dns` — DNS Lookups

Resolve A, AAAA, MX, NS, and TXT records for a domain, or perform a reverse lookup for an IP address.

```bash
trident dns example.com
trident dns 8.8.8.8
```

#### `asn` — ASN Lookup

Look up ASN information for an IP address or ASN number via Team Cymru DNS.

```bash
trident asn 8.8.8.8
trident asn AS15169
trident asn 2001:4860:4860::8888
```

#### `crtsh` — Certificate Transparency

Search [crt.sh](https://crt.sh) certificate transparency logs for subdomains of a domain.

```bash
trident crtsh example.com
```

### Output Formats

**Text (default):**
```bash
trident dns example.com
trident asn AS15169 -o text
```

**JSON:**
```bash
trident dns example.com -o json
trident crtsh example.com -o json
```

## Configuration

The config file is created automatically at `~/.config/trident/config.yaml` (Linux/macOS) or `%APPDATA%\trident\config.yaml` (Windows) with `0600` permissions.

Environment variables override config file values. All variables use the `TRIDENT_` prefix:

```bash
TRIDENT_OUTPUT=json trident dns example.com
TRIDENT_VERBOSE=true trident asn AS15169
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
go test ./internal/services/... -run TestCrtshService -v

# Lint
golangci-lint run
```

### Project Structure

```
cmd/trident/        # Entry point — delegates to cli.Execute()
internal/
  cli/              # Cobra command tree and output wiring
  config/           # Viper config loading
  services/         # One package per OSINT service
    dns/            # DNS record lookups (net package)
    asn/            # ASN lookups via Team Cymru DNS
    crtsh/          # Certificate transparency via crt.sh
  output/           # Text (tablewriter) and JSON formatters
  validate/         # Shared input validators
```

## Roadmap

- **Phase 2:** Stdin/bulk input, concurrency, proxy support, PAP labeling, defanging, ThreatMiner, PGP
- **Phase 3:** GoReleaser distribution, SBOM, Cosign signing, rate limiting
- **Phase 4:** Caching, Quad9, Tor, Robtex, Umbrella services
- **Phase 5:** 40+ API-key services (Shodan, VirusTotal, Censys, etc.)

## License

See [LICENSE](LICENSE).
