# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Trident** is a Go-based OSINT CLI tool (port of Python's [Harpoon](https://github.com/Te-k/harpoon)). It is currently **greenfield** — the PRD is in `docs/PRD.md` but no source code exists yet. The MVP (Phase 1) delivers three keyless reconnaissance services: DNS, ASN, and crt.sh.

## Commands

Once the Go module is initialized, these commands apply:

```bash
# Build
go build ./...

# Run all tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# Run a single test
go test ./internal/services/... -run TestCrtshService -v

# Lint (strict)
golangci-lint run

# Run the CLI
go run ./cmd/trident/main.go dns example.com
go run ./cmd/trident/main.go asn AS15169
go run ./cmd/trident/main.go crtsh example.com
```

## Architecture

### Directory Structure (to implement)

```
cmd/trident/        # main.go — delegates immediately to run()
internal/
  cli/              # Cobra root command, global flags, output formatting
  config/           # Viper config loading (~/.config/trident/config.yaml)
  services/         # One package per service (dns/, asn/, crtsh/)
  output/           # Text (tablewriter) and JSON formatters
```

### Core Patterns

**Dependency Injection:** Constructor injection everywhere. No global state or singletons.
```go
func NewCrtshService(client HttpClientInterface, logger *slog.Logger) *CrtshService
```

**`run` function pattern:** `main()` delegates to `run()` which accepts all dependencies and returns an error — enables testability.

**Interfaces for all external interactions:**
```go
type HttpClientInterface interface { ... }
type DNSResolverInterface interface { ... }
```
These interfaces are in `internal/services/` and mocked in tests via `jarcoal/httpmock` (HTTP) and custom resolver mocks (DNS).

**Service interface** — every service implements:
```go
type Service interface {
    Name() string
    Run(ctx context.Context, input string) (Result, error)
}
```

### Service Implementations (Phase 1)

| Command | Implementation | PAP |
|---------|---------------|-----|
| `dns` | Go `net` package — A, AAAA, MX, NS, TXT records | GREEN |
| `asn` | Team Cymru DNS TXT records via `net.Resolver` (format: `<reversed-ip>.origin.asn.cymru.com`) — no `os/exec` | AMBER |
| `crtsh` | HTTP GET `https://crt.sh/?q=%.<domain>&output=json` via `imroc/req` | AMBER |

### Configuration

- File: `~/.config/trident/config.yaml` (created with `0600` permissions)
- Managed via `spf13/viper`
- Env vars take precedence; prefix: `TRIDENT_*`
- Respect XDG on Linux, AppData on Windows

### Global Flags (Phase 1)

| Flag | Default |
|------|---------|
| `--config` | `~/.config/trident/config.yaml` |
| `--verbose` / `-v` | Info level logging |
| `--output` / `-o` | `text` (also: `json`) |

### Tech Stack

- **CLI:** `spf13/cobra`
- **Config:** `spf13/viper`
- **HTTP:** `imroc/req` v3 (no external SDKs — all APIs implemented natively)
- **Logging:** `log/slog` (stdlib only — no zap/logrus)
- **Tables:** `olekukonko/tablewriter`
- **Tests:** `stretchr/testify` + `jarcoal/httpmock`
- **Lint:** `golangci-lint` (strict — CI fails on any lint error)

## Key Constraints

- **No `os/exec`** for DNS — use `net.Resolver` directly
- **Enforced HTTPS only** — no `InsecureSkipVerify`
- **Output sanitization** — strip ANSI escape sequences from external data before printing
- **80% minimum test coverage** — integration tests use recorded HTTP fixtures
- **Cross-platform** — must compile on Linux, macOS, Windows; use `filepath.Join`

## Phase Roadmap

The full PRD is in `docs/PRD.md`. Phases deferred from MVP:
- **Phase 2:** Stdin/bulk input, concurrency (`--concurrency`), proxy (`--proxy`), PAP system, defanging, ThreatMiner, PGP
- **Phase 3:** GoReleaser, SBOM (CycloneDX), Cosign signing, rate limiting with jitter, `burn` command
- **Phase 4:** cache, quad9, tor, robtex, umbrella services
- **Phase 5:** 40+ API-key services (Shodan, VirusTotal, Censys, etc.)
- **Phase 6+:** TLS fingerprint evasion, honeypot detection, encrypted workspace
