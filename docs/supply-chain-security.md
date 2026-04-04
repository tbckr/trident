# Supply Chain Security

This document describes the supply chain hardening measures in place for trident. The goal is to protect users from tampered binaries, compromised dependencies, and unauthorized changes to the repository.

## Overview

| Layer | Measure | Status |
|-------|---------|--------|
| Repository | Branch & tag rulesets, CODEOWNERS, action restrictions, immutable releases | `harden-repo.sh` |
| CI | SHA-pinned actions, least-privilege permissions, sandboxed steps | `ci.yml` |
| Dependencies | govulncheck (daily + per-PR), Dependabot, license audit | `ci.yml`, `vuln-schedule.yml` |
| Release | Reproducible builds, SBOM, VEX, GitHub Artifact Attestation | `release.yml` |
| Monitoring | CodeQL SAST, OpenSSF Scorecard, secret scanning, push protection, tool version checks | `codeql.yml`, `scorecard.yml`, `tool-versions.yml`, repo settings |

## Repository Settings

Applied via [`scripts/harden-repo.sh`](../scripts/harden-repo.sh) (idempotent, requires `gh` CLI with admin access):

- **Wiki & Projects** disabled (reduces attack surface for social engineering).
- **Auto-merge** disabled (all merges require manual action).
- **Delete branch on merge** enabled (prevents stale branch accumulation).
- **Immutable releases** enabled (tags locked to commit after publication; assets protected from modification or deletion).
- **CODEOWNERS** (`.github/CODEOWNERS`) enforces code review routing — all files require review from the repository owner.

## Branch Rulesets (`main`)

Branch protection uses two GitHub Rulesets (migrated from legacy branch protection):

### `main-branch-integrity` (no bypass — enforced for everyone, including admins)

- **Signed commits** required.
- **Linear history** required (no merge commits).

### `main-branch-protection` (admin bypass)

- **Required status checks** (strict, pinned to GitHub Actions `app_id: 15368`): Test, Lint, Vulnerability Check, License Check, GoReleaser Lint, Nix Flake Check.
- **PR reviews**: 1 approval required, stale reviews dismissed on push, code owner review required, last push approval required.
- **Conversation resolution** required before merge.
- **Merge methods**: rebase and squash only.
- **Force push and branch deletion** blocked.

The legacy branch protection rule is removed automatically by `harden-repo.sh`.

## Tag Rulesets

### `release-tag-protection` (no bypass — enforced for everyone, including admins)

Protects all tags matching `v*`:

- **Tag deletion** blocked.
- **Tag overwrite** (non-fast-forward) blocked.
- **Signed tags** required.

## GitHub Actions Hardening

### Permission Model

All workflows use **least-privilege `permissions`** blocks:

- `ci.yml`: `contents: read` only.
- `release.yml`: `contents: write` + `id-token: write` + `attestations: write` (needed for GitHub Artifact Attestation).
- `vuln-schedule.yml`: `contents: read` only.
- `scorecard.yml`: `read-all` at workflow level; `security-events: write` + `id-token: write` at job level.
- `codeql.yml`: `contents: read` at workflow level; `security-events: write` at job level.
- `tool-versions.yml`: `contents: read` at workflow level; `issues: write` at job level.

### Action Restrictions

Repository-level policy (via `harden-repo.sh`):

- **GitHub-owned actions**: allowed.
- **Verified marketplace actions**: allowed.
- **Third-party actions**: explicitly allowlisted only:
  - `goreleaser/goreleaser-action`
  - `cachix/install-nix-action`
  - `golangci/golangci-lint-action`
  - `geomys/sandboxed-step`
  - `anchore/sbom-action`
  - `ossf/scorecard-action`
- **Fork PR workflows**: require approval before running.

### SHA Pinning

All `uses:` references are pinned to full commit SHAs (not tags). Dependabot is configured for `github-actions` to propose SHA updates via PRs.

### Sandboxed Steps

Security-sensitive CI steps (govulncheck, license checks) run inside [`geomys/sandboxed-step`](https://github.com/geomys/sandboxed-step), which isolates execution from the runner environment.

### Credential Hygiene

All checkout steps use `persist-credentials: false` to prevent the `GITHUB_TOKEN` from leaking into subsequent steps or being captured by compromised dependencies.

## Dependency Management

### Vulnerability Scanning

- **Per-PR/push**: `govulncheck ./...` runs in CI on every code change.
- **Daily schedule**: `vuln-schedule.yml` runs govulncheck at 06:00 UTC every day to catch newly disclosed vulnerabilities.
- **Dependabot alerts**: enabled for the repository.

govulncheck performs reachability analysis — it only flags vulnerabilities in code paths actually called by trident, reducing false positives compared to SCA scanners.

### Dependency Updates

- **GitHub Actions**: Dependabot monitors and proposes SHA-pinned updates.
- **Go modules**: `latest-deps.yml` runs weekly, upgrading direct dependencies only (`go get <pkg>@latest`, never `go get -u` which would upgrade all transitive deps).
- **Go modules are NOT managed by Dependabot** — govulncheck handles reachability-aware vulnerability detection, and `latest-deps.yml` handles freshness.

### License Compliance

`go-licenses check` runs in CI against an allowlist of OSI-approved licenses (MIT, Apache-2.0, BSD-2-Clause, BSD-3-Clause, ISC, MPL-2.0, GPL-3.0, GPL-3.0-only). Unapproved licenses fail the build.

### Module Verification

`go mod verify` runs in CI to ensure downloaded modules match `go.sum` checksums. `go mod tidy` + `git diff --exit-code` ensures no untracked dependency changes slip through.

## Release Pipeline

### Build Reproducibility

GoReleaser is configured for reproducible builds:

- `CGO_ENABLED=0` (static binaries, no system library variance).
- `-trimpath` (strips local filesystem paths from binaries).
- `-s -w` (strips debug symbols).
- `mod_timestamp: "{{ .CommitTimestamp }}"` (deterministic file timestamps).
- `gomod.proxy: true` (fetches modules via the Go module proxy for integrity).
- Source archive included (`source.enabled: true`).

### Software Bill of Materials (SBOM)

Every release includes a CycloneDX SBOM (`trident-<version>.cdx.json`) generated by [Syft](https://github.com/anchore/syft) from the source archive. This allows consumers to inventory all transitive dependencies.

### Vulnerability Exchange (VEX)

Every release includes an [OpenVEX](https://openvex.dev/) document (`trident-<version>.openvex.json`) generated by [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) (`-format openvex`). This document records the vulnerability status of all known CVEs against the release at build time:

- **`not_affected`** (with justification `vulnerable_code_not_in_execute_path`) — the vulnerable code exists in a dependency but is not reachable from trident's call graph.
- **`affected`** — the vulnerability is reachable.

VEX complements the SBOM: the SBOM lists *what* dependencies are included, while the VEX document explains *whether* known vulnerabilities actually affect trident. This is powered by govulncheck's reachability analysis.

The VEX document is included in `checksums.txt` and attested alongside all other release artifacts.

### Build Provenance

Each release is attested via [GitHub Artifact Attestation](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds) using [`actions/attest-build-provenance`](https://github.com/actions/attest-build-provenance):

- Attests every artifact listed in `checksums.txt`, proving it was built by the official release workflow.
- Attestations are signed automatically by GitHub and stored in the GitHub Attestation API.
- Verifiable with `gh attestation verify` — no external tools (like cosign) required.

### Release Signing Flow

1. GoReleaser builds binaries, generates the VEX document and SBOM, generates `checksums.txt` (including hashes for VEX and SBOM), and creates a **draft** release.
2. `actions/attest-build-provenance` attests every artifact listed in `checksums.txt` via GitHub Artifact Attestation.
3. The release is promoted from draft to published.
4. **Immutable releases** lock the tag and all assets against modification or deletion.

### Verifying a Release

Users can verify any release artifact using [`scripts/verify-release.sh`](../scripts/verify-release.sh):

```bash
# Download your archive first, then:
./scripts/verify-release.sh trident_Linux_x86_64.tar.gz
```

This script verifies the build provenance attestation with `gh attestation verify`, which proves both provenance (built by the official release workflow) and integrity (SHA-256 digest match).

Requirements: [GitHub CLI](https://cli.github.com/) (2.49+).

## Tool Version Monitoring

Go tools pinned in workflow files (govulncheck, go-licenses, golangci-lint, goreleaser) are not covered by Dependabot. The `tool-versions.yml` workflow runs weekly (Mondays at 06:00 UTC) and uses [`scripts/check-tool-versions.sh`](../scripts/check-tool-versions.sh) to compare pinned versions against the latest releases (via the Go module proxy and GitHub API). When updates are available, it creates or updates a GitHub issue with a summary table.

The script parses versions directly from workflow files — there is no separate version manifest to keep in sync.

## CodeQL SAST Analysis

[CodeQL](https://codeql.github.com/) runs on every push and pull request to `main`, plus weekly (Monday 06:00 UTC) via `codeql.yml`. It performs semantic code analysis for Go, detecting security vulnerabilities such as injection flaws, path traversals, and unsafe operations. Results are uploaded to GitHub's Security tab as SARIF reports.

## OpenSSF Scorecard

The [OpenSSF Scorecard](https://securityscorecards.dev/viewer/?uri=github.com/tbckr/trident) runs weekly (Mondays at 06:00 UTC) via `scorecard.yml` and uploads SARIF results to GitHub's Security tab. This provides continuous, independent assessment of the project's security posture across dimensions like branch protection, dependency management, and CI practices.

## Security Reporting

- **Secret scanning** with push protection is enabled (blocks commits containing detected secrets).
- **Private vulnerability reporting** is enabled — security researchers can report vulnerabilities confidentially via GitHub's built-in mechanism.

See [`SECURITY.md`](../SECURITY.md) for the project's security policy.

## Hardening Script

To apply or re-apply all repository-level settings:

```bash
# Requires gh CLI authenticated with admin access
./scripts/harden-repo.sh              # current repo
./scripts/harden-repo.sh owner/repo   # specific repo
```

The script is idempotent and includes retry logic for transient GitHub API failures (rate limiting, connection issues).