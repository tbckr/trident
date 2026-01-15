# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Trident is a Go-based OSINT (Open Source Intelligence) CLI tool, a port of the Python tool Harpoon. It provides automated querying of threat intelligence, network, and identity platforms with a focus on operational security (OpSec).

**Phase 1 Focus**: Core CLI framework + 5 keyless services (DNS, ASN, crt.sh, ThreatMiner, PGP).

## Commands

```bash
# Build
go build -o bin/trident ./cmd/trident

# Test with coverage (80% minimum required)
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | grep total

# Lint (must pass, build fails on errors)
golangci-lint run

# Security scanning
gosec ./...
govulncheck ./...
```

## Architecture

### Core Patterns

**The run Function Pattern**: `main()` must be ultra-simple - context, logging setup, call `run()`, handle exit. All dependencies injected into `run()`.

```go
func run(
    ctx context.Context,
    args []string,
    getenv func(string) string,
    stdin io.Reader,
    stdout, stderr io.Writer,
    logger *slog.Logger,
    levelVar *slog.LevelVar,
) error
```

**No Global State**: Global variables are forbidden - no package-level Cobra commands, no `init()` for flags. Use constructors: `NewRootCmd(logger, levelVar)`.

**Dependency Injection**: Services accept dependencies via interfaces. Example: `NewThreatMinerService(client HttpClientInterface, logger *slog.Logger)`.

### Standard Libraries (Pre-approved)

- CLI: `github.com/spf13/cobra`
- Config: `github.com/spf13/viper` (config at `~/.config/trident/config.yaml`)
- HTTP: `github.com/imroc/req/v3` (no external SDKs - implement raw HTTP)
- Logging: `log/slog` (stdlib only)
- Testing: `github.com/stretchr/testify`, `github.com/jarcoal/httpmock`
- Output: `github.com/olekukonko/tablewriter`

### Testing Requirements

- **Black-box testing**: Tests in separate package (`pkg_test`, not `pkg`)
- **Table-driven tests**: Standard pattern with `t.Run()`
- **80% coverage mandatory**
- **HTTP mocking**: Use `httpmock` for recorded responses, no real API calls in tests

### Key Global Flags

`--proxy` (HTTP/HTTPS/SOCKS5), `--user-agent`, `--pap-limit` (RED/AMBER/GREEN/WHITE), `--defang`/`--no-defang`, `--concurrency`, `--output` (text/json/plain)

### PAP (Permissible Actions Protocol)

Commands have assigned PAP levels controlling execution based on `--pap-limit`:
- **RED**: Offline/local only
- **AMBER**: Third-party APIs (crt.sh, ThreatMiner, PGP, ASN)
- **GREEN**: Direct target interaction (DNS)
- **WHITE**: Unrestricted

### Input Handling

Commands accept args OR stdin (one entry per line). Args take priority. Example: `cat domains.txt | trident dns`

## Code Style

- Accept interfaces, return structs
- Small interfaces (1-3 methods) defined where used
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Guard clauses over nested if/else
- Modern Go (1.21+): use `any`, `slices`/`maps` packages, `min`/`max`
- Conventional commits: `feat:`, `fix:`, `refactor:`, etc.

## Vibe

Dry, concise communication. Comments explain *why*, not *what*. Sparse humor in comments is fine if it lands. Cursing in comments is allowed within reason. When stuck, search official docs before pivoting. Leave the repo better than you found it.
