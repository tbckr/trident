# Build all packages
build:
    go build ./...

# Run all tests with coverage
test:
    go test ./... -coverprofile=coverage.out -covermode=atomic

# Run tests for a specific package (e.g., just test-pkg ./internal/services/dns/...)
test-pkg pkg:
    go test {{ pkg }} -v

# Check service coverage meets 80% threshold
coverage:
    #!/usr/bin/env bash
    set -euo pipefail
    go test ./internal/services/... -coverprofile=svc_coverage.out -covermode=atomic
    total=$(go tool cover -func=svc_coverage.out | grep ^total | awk '{print $3}' | tr -d '%')
    echo "Service coverage: ${total}%"
    awk "BEGIN { if ($total+0 < 80) { print \"FAIL: coverage \" $total \"% < 80%\"; exit 1 } }"

# Run linter
lint:
    golangci-lint run

# Tidy and verify modules
tidy:
    go mod tidy
    go mod verify

# Check that go.mod/go.sum are tidy (fails if dirty)
tidy-check:
    #!/usr/bin/env bash
    set -euo pipefail
    go mod tidy
    go mod verify
    git diff --exit-code go.mod go.sum

# Run govulncheck
vuln:
    go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Check dependency licenses
license-check:
    go run github.com/google/go-licenses/v2@latest check github.com/tbckr/trident/... \
        --allowed_licenses=MIT,Apache-2.0,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0,GPL-3.0,GPL-3.0-only

# Build the Nix package locally
flake-build:
    nix build

# Run Nix flake check
flake-check:
    nix flake check

# Run all CI checks locally
ci: tidy-check build test coverage lint vuln license-check flake-check

# Release: tag next version with svu and push
release:
    #!/usr/bin/env bash
    set -euo pipefail
    next=$(svu next)
    git tag "${next}"
    git push
    git push --tags
    @echo "Released ${next}"

# Validate .goreleaser.yaml config
goreleaser-check:
    goreleaser check

# Upgrade all direct dependencies and run tests (mirrors latest-deps.yml)
upgrade-deps:
    ./scripts/upgrade-deps.sh
    go test ./...

# Update Nix flake inputs
flake-update:
    nix flake update
