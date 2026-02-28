# Contributing to trident

Thank you for your interest in contributing to trident! This guide will help you get started.

## Getting Started

### Prerequisites

- Go 1.26 or later
- golangci-lint v2

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

A [justfile](https://github.com/casey/just) wraps common tasks. Install `just` and run:

```bash
just build       # Build all packages
just test        # Run all tests with coverage
just lint        # Run golangci-lint
just tidy        # Tidy and verify modules
just ci          # Run all CI checks locally (build, test, coverage, lint, vuln, license, flake)
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

## Reporting Bugs

Open a [bug report](https://github.com/tbckr/trident/issues/new?template=bug_report.yml) using the issue template. The form will guide you through the required information.

## Responsible Use

trident is an OSINT tool intended for legitimate security research and authorized testing. Please review the [Responsible Use](README.md#responsible-use) section in the README before contributing new features.
