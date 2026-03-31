# Supply Chain Security

This document describes the supply chain hardening measures in place for trident. The goal is to protect users from tampered binaries, compromised dependencies, and unauthorized changes to the repository.

## Overview

| Layer | Measure | Status |
|-------|---------|--------|
| Repository | Branch protection, action restrictions, immutable releases | `harden-repo.sh` |
| CI | SHA-pinned actions, least-privilege permissions, sandboxed steps | `ci.yml` |
| Dependencies | govulncheck (daily + per-PR), Dependabot, license audit | `ci.yml`, `vuln-schedule.yml` |
| Release | Reproducible builds, SBOM, SLSA provenance, Cosign signing | `release.yml` |
| Monitoring | OpenSSF Scorecard, secret scanning, push protection, tool version checks | `scorecard.yml`, `tool-versions.yml`, repo settings |

## Repository Settings

Applied via [`scripts/harden-repo.sh`](../scripts/harden-repo.sh) (idempotent, requires `gh` CLI with admin access):

- **Wiki & Projects** disabled (reduces attack surface for social engineering).
- **Auto-merge** disabled (all merges require manual action).
- **Delete branch on merge** enabled (prevents stale branch accumulation).
- **Immutable releases** enabled (tags locked to commit after publication; assets protected from modification or deletion).

## Branch Protection (`main`)

- **Required status checks** (strict): Test, Lint, Vulnerability Check, License Check, Nix Flake Check.
- **Enforce admins**: yes (no bypass, even for repository owners).
- **PR reviews required** with stale review dismissal.
- **Linear history** required (no merge commits).
- **Force push and branch deletion** blocked.
- **Conversation resolution** required before merge.

## GitHub Actions Hardening

### Permission Model

All workflows use **least-privilege `permissions`** blocks:

- `ci.yml`: `contents: read` only.
- `release.yml`: `contents: write` + `id-token: write` (needed for Cosign keyless signing).
- `vuln-schedule.yml`: `contents: read` only.
- `scorecard.yml`: `read-all` at workflow level; `security-events: write` + `id-token: write` at job level.

### Action Restrictions

Repository-level policy (via `harden-repo.sh`):

- **GitHub-owned actions**: allowed.
- **Verified marketplace actions**: allowed.
- **Third-party actions**: explicitly allowlisted only:
  - `goreleaser/goreleaser-action`
  - `cachix/install-nix-action`
  - `golangci/golangci-lint-action`
  - `geomys/sandboxed-step`
  - `sigstore/cosign-installer`
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

Every release includes a CycloneDX SBOM (`*.cdx.json`) generated by [Syft](https://github.com/anchore/syft) from the source archive. This allows consumers to inventory all transitive dependencies.

### SLSA Provenance

Each release generates a [SLSA Provenance v1](https://slsa.dev/provenance/v1) predicate via [`scripts/generate-provenance.sh`](../scripts/generate-provenance.sh):

- Records the exact commit, workflow, runner OS/architecture, and build timestamps.
- Signed with Cosign keyless signing (Sigstore OIDC) and attached to the release as `checksums.txt.slsa-provenance.sigstore.json`.
- Certificate identity is bound to the release workflow: `https://github.com/tbckr/trident/.github/workflows/release.yml@refs/tags/<VERSION>`.

### Release Signing Flow

1. GoReleaser builds binaries, generates `checksums.txt`, and creates a **draft** release.
2. `generate-provenance.sh` creates the SLSA predicate from CI environment variables.
3. `cosign attest-blob` signs `checksums.txt` with the provenance predicate (keyless OIDC).
4. The provenance bundle is uploaded to the release.
5. The release is promoted from draft to published.
6. **Immutable releases** lock the tag and all assets against modification or deletion.

### Verifying a Release

Users can verify any release artifact using [`scripts/verify-release.sh`](../scripts/verify-release.sh):

```bash
# Download your archive first, then:
./scripts/verify-release.sh v0.5.0 trident_Linux_x86_64.tar.gz
```

This script:

1. Downloads `checksums.txt` and the provenance bundle from the release.
2. Verifies the SLSA provenance attestation with `cosign verify-blob-attestation`.
3. Validates the archive's SHA-256 checksum against `checksums.txt`.

Requirements: [cosign v2+](https://docs.sigstore.dev/cosign/system_config/installation/) and `curl`.

## Tool Version Monitoring

Go tools pinned in workflow files (govulncheck, go-licenses, golangci-lint, goreleaser) are not covered by Dependabot. The `tool-versions.yml` workflow runs weekly (Mondays at 06:00 UTC) and uses [`scripts/check-tool-versions.sh`](../scripts/check-tool-versions.sh) to compare pinned versions against the latest releases (via the Go module proxy and GitHub API). When updates are available, it creates or updates a GitHub issue with a summary table.

The script parses versions directly from workflow files — there is no separate version manifest to keep in sync.

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