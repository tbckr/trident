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
just outdated           # List newer dependencies, tools and images to review
just upgrade            # Upgrade direct dependencies (see Updating Dependencies)
just harden-repo        # Apply repository hardening settings
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
| Weekly Mon 06:00 UTC | `latest-deps.yml` | Tests against the latest direct dependencies, checks Go release support, lists versions to review |
| Weekly Mon 06:00 UTC | `codeql.yml` | CodeQL SAST analysis for Go |

## Updating Dependencies

There is no update bot for Go modules or for the tools pinned in the workflows. They are upgraded on the project's own schedule:

```bash
just outdated                    # what has a newer release: direct modules, Go, pinned tools and images, flake inputs
just upgrade                     # upgrade the direct Go modules
git add go.mod go.sum flake.nix  # after updating vendorHash; tidy-check compares against the index
just ci
```

`just upgrade` changes `go.sum`, so update `vendorHash` in `flake.nix` as well (see `CLAUDE.md`, Nix Flake). Commit the result as a single `chore(deps):` commit, or `fix(deps):` if it fixes a vulnerability that govulncheck reports for trident, so that the next release ships it.

**Only direct dependencies are upgraded.** A transitive dependency changes only where an upgraded direct dependency requires it; keeping its own dependencies current is that dependency's job. `just upgrade` ([`scripts/deps.sh`](scripts/deps.sh)) runs `go get <module>@latest` for each direct module and `go mod tidy`. Never use `go get -u`: it raises transitive versions on its own. If a vulnerability needs a transitive module raised before its direct dependency ships a fix, raise only that module, to the fixed version and not to the latest.

**Held modules.** A direct module that has to stay at an older version carries a `// Revisit ...` note on its `require` line in `go.mod` saying why and until when, so `just outdated` lists it under Reminders. `just upgrade` does not know about holds and raises it like any other module, so set it back with `go get <module>@<version>` before committing. If the newer release needs a later Go, `go get` may also switch toolchains and raise the go directive: check that the `go` line in `go.mod` did not move.

**Major versions** of direct modules are a manual step that `just outdated` does not detect: a new major is a new module path, and `go list -m -u` only looks within the current one. Check them by hand when upgrading, for example with `go list -m -f '{{.Version}}' <module>/v<N+1>@latest`.

**Manual steps.** `just outdated` lists them for review on every upgrade; it never fails:

- The Go release in `go.mod`: only the two newest Go releases get security fixes, and `latest-deps.yml` fails once ours drops out. `go get` also raises it where a new release of a direct dependency requires a later Go, which `just upgrade` then needs to run with. Raise the release in `.go-version` and `buildGo1XXModule` and `go_1_XX` in `flake.nix` with it. `.go-version` holds only the release (`1.26`), never a patch. Every `actions/setup-go` step reads it, except the `go` job of `latest-deps.yml`, which runs on the newest stable Go on purpose; the steps that build, lint, scan and release add `check-latest`, so they run on the newest patch of that release. `go.mod` cannot serve for this, because `go get` writes a patch level into the go directive (`go 1.26.0`) as soon as a dependency declares one, and setup-go then installs exactly that patch.
- golangci-lint (`GOLANGCI_LINT_VERSION` in `ci.yml`). Keep it at the version of the dev shell, which takes golangci-lint from nixpkgs, or CI and `just lint` lint with different rules.
- GoReleaser (`GORELEASER_VERSION` in `release.yml` and `goreleaser-lint.yml`), an exact version because it runs in the job that publishes and attests.
- govulncheck and go-licenses (`go run <module>@vX.Y.Z` in `ci.yml`, `vuln-schedule.yml` and `.goreleaser.yaml`).
- The semgrep and betterleaks container images (`semgrep.yml`, `betterleaks.yml`), pinned by tag and digest. Bump both together to the pin that `just outdated` prints. A tag reported as rebuilt kept its version but now has another digest upstream.
- The flake inputs (`just flake-update`), shown with the age of their lock.

Every pin of a tool carries the same version; `just outdated` reports pins that disagree. When bumping a tool, change only its version, never the SHA of a `uses:` line or its `# vX.Y.Z` comment, which Dependabot maintains.

A note in code or configuration that waits for an event (a fix upstream, a new release) says "Revisit" or "Drop this once", so that `just outdated` lists it under Reminders.

**Automation:**

- `latest-deps.yml`, weekly: tests against the newest direct dependencies, upgraded exactly like `just upgrade` does with the newest Go, inside a gVisor sandbox, and checks that the Go release in `go.mod` still gets security fixes. It commits nothing; a failure means the next upgrade needs work. The list of `just outdated` goes into the run's summary.
- `vuln-schedule.yml`, daily, and `ci.yml`, on every change: govulncheck.
- Dependabot only bumps the SHA-pinned GitHub Actions, monthly in one pull request, and leaves out releases younger than seven days.

## Security Practices for Contributors

When modifying CI workflows:

- All `uses:` references must be **SHA-pinned** (not tag-pinned) — Dependabot proposes updates monthly
- Checkout steps must use `persist-credentials: false`
- Workflow `permissions` must follow least-privilege (start with `contents: read`)
- Security-sensitive steps (govulncheck, license checks) should use [`geomys/sandboxed-step`](https://github.com/geomys/sandboxed-step)
- Never use `go get -u` — it upgrades all transitive dependencies; use `go get <pkg>@latest` for direct deps only

## Reporting Bugs

Open a [bug report](https://github.com/tbckr/trident/issues/new?template=bug_report.yml) using the issue template. The form will guide you through the required information.

## Responsible Use

trident is an OSINT tool intended for legitimate security research and authorized testing. Please review the [Responsible Use](README.md#responsible-use) section in the README before contributing new features.
