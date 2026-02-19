# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Trident** is a Go-based OSINT CLI tool (port of Python's [Harpoon](https://github.com/Te-k/harpoon)). Phase 1 MVP is implemented — three keyless OSINT services: DNS, ASN, and crt.sh. The PRD is in `docs/PRD.md`.

## Tools

**Library documentation:** Always use the context7 MCP (`mcp__plugin_context7-plugin_context7__resolve-library-id` + `mcp__plugin_context7-plugin_context7__query-docs`) to look up library docs. Never guess API shapes — fetch authoritative documentation first.

## Module
`github.com/tbckr/trident`

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

### Directory Structure

```
cmd/trident/        # main.go — delegates immediately to run()
internal/
  cli/              # Cobra root command, global flags, output formatting
  config/           # Viper config loading (~/.config/trident/config.yaml)
  services/         # One package per service (dns/, asn/, crtsh/)
  output/           # Text (tablewriter) and JSON formatters
  validate/         # Shared input validators — IsDomain() and future helpers
```

### Core Patterns

**Dependency Injection:** Constructor injection everywhere. No global state or singletons.
```go
func NewDNSService(resolver DNSResolverInterface, logger *slog.Logger) *DNSService
func NewCrtshService(client *req.Client, logger *slog.Logger) *CrtshService
```

**`*req.Client` is a hard dependency** — not abstracted behind an interface. Mock HTTP in tests via `httpmock.ActivateNonDefault(client.GetClient())`.

**tablewriter v1.1.3 API:** `table.Header([]string{...})` + `table.Bulk([][]string{...})` + `table.Render()`. Old `SetHeader`/`Append([]string)` don't exist — use `Bulk` for multi-row, `Append(any)` for single row.

**tablewriter error returns:** Both `table.Bulk(rows)` and `table.Render()` return `error` — always propagate them; `errcheck` will fail the lint if ignored.

**`output.NewWrappingTable`** — shared factory in `internal/output/terminal.go`; always use this in service `WriteText` methods instead of calling `tablewriter.NewTable` directly. Overhead values: 20 for 2-column tables (Type/Field + Value), 6 for 1-column tables.

**gosec G115 (`uintptr→int`)** — `term.GetSize(int(f.Fd()))` always triggers G115. Suppress with `//nolint:gosec // uintptr→int is safe for file descriptors; they fit in int on all supported platforms`.

**`internal/testutil`** — `MockResolver` (implements `DNSResolverInterface` with optional `*Fn` fields) + `NopLogger()`. Import in `_test` files for DNS/ASN service tests.

**crtsh URL:** Use `"%%.%s"` (double `%%`) in the constant so `fmt.Sprintf` emits a literal `%.` before the domain. `"%.%s"` silently causes an arg-count mismatch.

**`DNSResolverInterface`** — only DNS/ASN use an interface (for `*net.Resolver` mocking). Defined in `internal/services/interfaces.go`.

**`run` function pattern:** `main()` delegates to `run()` which accepts all dependencies and returns an error — enables testability.

**`services.ErrInvalidInput`** — unified validation sentinel in `internal/services/service.go`. New services must use this (not define their own `ErrInvalidInput`). Wrap with context: `fmt.Errorf("%w: must be …: %q", services.ErrInvalidInput, input)`.

**Type assertions in tests** — always use two-value form: `result, ok := raw.(*T); require.True(t, ok, "expected *T")`. Bare `raw.(*T)` panics on failure.

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
| `asn` | Team Cymru DNS: IPv4 → `<reversed>.origin.asn.cymru.com`; IPv6 → 32-nibble reversal + `.origin6.asn.cymru.com`; ASN → `AS<n>.asn.cymru.com` | AMBER |
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
- **Lint:** `golangci-lint` v2 (strict — CI fails on any lint error). Config requires `version: "2"` at top; formatters (`gofmt`, `goimports`) go in `formatters:` section, not `linters:`. GitHub Action: `golangci/golangci-lint-action@v8` with `version: latest` (pinning a specific version risks Go version mismatch with `go.mod`).
- **golangci-lint v2 config structure:** `linters-settings` → `linters.settings`; `formatters-settings` → `formatters.settings`; `issues.exclude-rules` → `linters.exclusions.rules`. `goimports.local-prefixes` is an array (not a string). `gosimple` is merged into `staticcheck` — do not list it separately.
- **gosec suppressions:** `gosec.excludes` under `linters.settings` is unreliable; prefer `linters.exclusions.rules` with `text: "G304"` or an inline `//nolint:gosec // reason` comment. `nolintlint` will error if the directive is present but gosec doesn't fire on that line — remove unused nolint directives rather than suppressing them.
- **revive `package-comments`:** Every package must have a `// Package foo ...` comment in at least one file (primary file or `doc.go`). New packages without this will fail lint.

## Key Constraints

- **No external I/O in tests** — all DNS and HTTP must be mocked; no real network calls. DNS: `mockResolver` struct; HTTP: `httpmock.ActivateNonDefault(client.GetClient())`.
- **No `os/exec`** for DNS — use `net.Resolver` directly
- **Enforced HTTPS only** — no `InsecureSkipVerify`
- **Output sanitization** — strip ANSI escape sequences from external data before printing
- **80% minimum test coverage** — enforced on `./internal/services/...` only (CLI/cmd packages intentionally have 0%). CI uses `go test ./internal/services/... -coverprofile=svc_coverage.out`.
- **Cross-platform** — must compile on Linux, macOS, Windows; use `filepath.Join`

## Phase Roadmap

The full PRD is in `docs/PRD.md`. Phases deferred from MVP:
- **Phase 2:** Stdin/bulk input, concurrency (`--concurrency`), proxy (`--proxy`), PAP system, defanging, ThreatMiner, PGP
- **Phase 3:** GoReleaser, SBOM (CycloneDX), Cosign signing, rate limiting with jitter, `burn` command
- **Phase 4:** cache, quad9, tor, robtex, umbrella services
- **Phase 5:** 40+ API-key services (Shodan, VirusTotal, Censys, etc.)
- **Phase 6+:** TLS fingerprint evasion, honeypot detection, encrypted workspace
