# Contributing to trident

Thank you for your interest in contributing to trident! This guide will help you get started.

## Getting Started

### Prerequisites

- Go 1.26 or later
- [golangci-lint](https://golangci-lint.run/) v2
- [just](https://github.com/casey/just) (task runner)

Optional (for releases and Nix builds):
- [goreleaser](https://goreleaser.com/) v2
- [svu](https://github.com/caarlos0/svu) (semantic version utility)
- [Nix](https://nixos.org/) (for flake builds and dev shell)

### Nix Dev Shell

If you use [Nix](https://nixos.org/), `nix develop` provides Go, golangci-lint, goreleaser, and svu — no manual tool installation needed:

```bash
nix develop
```

### Setup

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/<your-username>/trident.git
   cd trident
   ```
3. Create a branch for your change:
   ```bash
   git checkout -b feat/my-change
   ```

## Development Workflow

A [justfile](https://github.com/casey/just) wraps common tasks:

**Build & Test:**

```bash
just build              # Build all packages
just test               # Run all tests with coverage
just test-pkg ./internal/services/dns/...  # Test a specific package
just test-race          # Run all tests with race detector
just fuzz ./internal/output/...            # Run fuzz tests for a package
just coverage           # Check service coverage meets 80% threshold
```

**Code Quality:**

```bash
just fmt                # Format all Go files with gofmt
just lint               # Run golangci-lint
just tidy               # Tidy and verify modules
just tidy-check         # Verify modules are tidy (fails if dirty)
just vuln               # Run govulncheck
just license-check      # Check dependency licenses against allowlist
```

**CI & Nix:**

```bash
just ci                 # Run all CI checks locally
just flake-build        # Build the Nix package locally
just flake-check        # Run Nix flake check
just flake-update       # Update Nix flake inputs
```

**Release & Maintenance:**

```bash
just release            # Tag next version with svu and push
just goreleaser-check   # Validate .goreleaser.yaml config
just verify-release v0.10.0 trident_Linux_x86_64.tar.gz  # Verify release artifact
just upgrade-deps       # Upgrade direct dependencies and run tests
just harden-repo        # Apply repository hardening settings
just check-tool-versions  # Check pinned tool versions for updates
```

Or use the underlying Go commands directly:

```bash
go build ./...
go test ./...
golangci-lint run
go mod tidy
```

All checks must pass before submitting a pull request. Run `just ci` to verify everything at once.

## Code Standards

- **Test coverage**: 80% minimum on `./internal/services/...`
- **No external I/O in tests**: mock HTTP with `httpmock` and DNS with `testutil.MockResolver`
- **No mutation**: prefer immutable patterns — create new objects rather than modifying existing ones
- **Small files**: aim for 200-400 lines, 800 max
- **Small functions**: under 50 lines
- **Fuzz tests**: consider adding fuzz tests for input-parsing functions (`just fuzz <pkg>`); existing fuzz targets cover defanging (`internal/output/`) and domain validation (`internal/services/`)

## Adding a New Service

Each service lives in its own package under `internal/services/<name>/` with a required file layout:

```
internal/services/<name>/
├── doc.go           # // Package <name> ... comment only
├── service.go       # Service struct, constructor, Name, PAP, Run, helpers
├── result.go        # Result struct + IsEmpty, WriteText, WriteTable
└── multi_result.go  # MultiResult + WriteTable (omit if no bulk path)
```

Test files: `service_test.go`, `result_test.go`, `multi_result_test.go`.

**Checklist for adding a service:**

- [ ] Implement the service following the file layout above
- [ ] Export package-level `Name` and `PAP` constants
- [ ] Add the service to `allServices()` in `internal/cli/services.go`
- [ ] Update README.md in 5 places: quickstart, services table, PAP table, commands reference, architecture tree
- [ ] Achieve 80% test coverage (`just coverage`)
- [ ] No external I/O in tests — mock HTTP and DNS
- [ ] Run `just ci` to verify all checks pass

## Commit Messages

Use conventional commit format:

```
<type>: <description>
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf`, `ci`

Examples:
- `feat: add whois service`
- `fix: handle nil response in crtsh`
- `docs: update README with new command`

## Pull Requests

- Open an issue first for significant changes to discuss the approach
- One logical change per PR
- All CI checks must pass (tests, lint, govulncheck, license check)
- Keep the PR description clear about what changed and why

## CI Workflows

**On every push/PR to main:**

| Check | Workflow | Description |
|-------|----------|-------------|
| Test | `ci.yml` | Build, test, coverage threshold |
| Lint | `ci.yml` | golangci-lint v2 (strict) |
| Vulnerability Check | `ci.yml` | govulncheck (sandboxed) |
| License Check | `ci.yml` | Dependency license allowlist |
| Nix Flake Check | `ci.yml` | Nix build reproducibility |
| GoReleaser Lint | `goreleaser-lint.yml` | `.goreleaser.yaml` validation (on config changes) |
| CodeQL | `codeql.yml` | SAST analysis for Go |

**Scheduled:**

| Schedule | Workflow | Description |
|----------|----------|-------------|
| Daily 06:00 UTC | `vuln-schedule.yml` | govulncheck for newly disclosed CVEs |
| Weekly Mon 06:00 UTC | `scorecard.yml` | OpenSSF Scorecard assessment |
| Weekly Mon 06:00 UTC | `latest-deps.yml` | Direct dependency freshness check |
| Weekly Mon 06:00 UTC | `tool-versions.yml` | Pinned Go tool version updates |
| Weekly Mon 06:00 UTC | `codeql.yml` | CodeQL SAST analysis for Go |

## Security Practices for Contributors

When modifying CI workflows:

- All `uses:` references must be **SHA-pinned** (not tag-pinned) — Dependabot proposes updates
- Checkout steps must use `persist-credentials: false`
- Workflow `permissions` must follow least-privilege (start with `contents: read`)
- Security-sensitive steps (govulncheck, license checks) should use [`geomys/sandboxed-step`](https://github.com/geomys/sandboxed-step)
- Never use `go get -u` — it upgrades all transitive dependencies; use `go get <pkg>@latest` for direct deps only

## Reporting Bugs

Open a [bug report](https://github.com/tbckr/trident/issues/new?template=bug_report.yml) using the issue template. The form will guide you through the required information.

## Responsible Use

trident is an OSINT tool intended for legitimate security research and authorized testing. Please review the [Responsible Use](README.md#responsible-use) section in the README before contributing new features.
