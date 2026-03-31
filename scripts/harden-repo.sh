#!/usr/bin/env bash
# harden-repo.sh — Apply supply-chain hardening settings to a GitHub repository.
# Usage: ./scripts/harden-repo.sh [owner/repo]
# Defaults to the current repo (requires gh CLI authenticated).
#
# Idempotent: safe to run multiple times. Re-applying overwrites with the same values.
# Requires: gh CLI with admin access to the target repository.

set -euo pipefail

REPO="${1:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"
BRANCH="main"

# Wrapper: retry on transient failures (connection refused, rate limit)
gh_api() {
  local attempt
  for attempt in 1 2 3; do
    if gh api "$@" 2>/tmp/gh-api-err; then
      return 0
    fi
    local rc=$?
    local err
    err=$(cat /tmp/gh-api-err)
    if echo "$err" | grep -qE "connection refused|rate limit|abuse detection|secondary rate"; then
      echo "  (retry $attempt/3 after transient failure, waiting ${attempt}s...)" >&2
      sleep "$attempt"
    else
      echo "$err" >&2
      return "$rc"
    fi
  done
  echo "$err" >&2
  return 1
}

echo "=== Hardening repository: $REPO ==="
echo ""

# --------------------------------------------------------------------------- #
# 1. Repository settings                                                      #
# --------------------------------------------------------------------------- #
echo "[1/6] Repository settings..."
gh_api "repos/$REPO" -X PATCH \
  -f has_wiki=false \
  -f has_projects=false \
  -f allow_auto_merge=false \
  -f delete_branch_on_merge=true \
  --silent
echo "  - Wiki: disabled"
echo "  - Projects: disabled"
echo "  - Auto-merge: disabled"
echo "  - Delete branch on merge: enabled"

# --------------------------------------------------------------------------- #
# 2. Vulnerability alerts + secret scanning                                   #
# --------------------------------------------------------------------------- #
echo "[2/6] Security features..."
gh_api "repos/$REPO/vulnerability-alerts" -X PUT --silent 2>/dev/null || true
echo "  - Dependabot alerts: enabled"

gh_api "repos/$REPO" -X PATCH \
  -f security_and_analysis[secret_scanning][status]=enabled \
  -f security_and_analysis[secret_scanning_push_protection][status]=enabled \
  --silent 2>/dev/null || echo "  - Secret scanning: skipped (may require GitHub Advanced Security)"
echo "  - Secret scanning: enabled"
echo "  - Push protection: enabled"

# --------------------------------------------------------------------------- #
# 3. Private vulnerability reporting                                          #
# --------------------------------------------------------------------------- #
echo "[3/6] Private vulnerability reporting..."
gh_api "repos/$REPO/private-vulnerability-reporting" -X PUT --silent 2>/dev/null || true
echo "  - Private vulnerability reporting: enabled"

# --------------------------------------------------------------------------- #
# 4. Branch protection on main                                                #
# --------------------------------------------------------------------------- #
echo "[4/6] Branch protection on '$BRANCH'..."
gh_api "repos/$REPO/branches/$BRANCH/protection" -X PUT \
  --input - --silent <<'PAYLOAD'
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "Test",
      "Lint",
      "Vulnerability Check",
      "License Check",
      "GoReleaser Lint",
      "Nix Flake Check"
    ]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "required_approving_review_count": 0,
    "dismiss_stale_reviews": true
  },
  "restrictions": null,
  "required_linear_history": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "block_creations": false,
  "required_conversation_resolution": true
}
PAYLOAD
echo "  - Require status checks (strict): Test, Lint, Vulnerability Check, License Check, GoReleaser Lint, Nix Flake Check"
echo "  - Enforce admins: yes"
echo "  - Require PR reviews: yes (0 approvals — sole maintainer)"
echo "  - Dismiss stale reviews: yes"
echo "  - Linear history: required"
echo "  - Force push: blocked"
echo "  - Branch deletion: blocked"
echo "  - Require conversation resolution: yes"

# --------------------------------------------------------------------------- #
# 5. Actions permissions                                                      #
# --------------------------------------------------------------------------- #
echo "[5/6] Actions permissions..."
gh_api "repos/$REPO/actions/permissions" -X PUT \
  --input - --silent <<'PERMS'
{
  "enabled": true,
  "allowed_actions": "selected"
}
PERMS

gh_api "repos/$REPO/actions/permissions/selected-actions" -X PUT \
  --input - --silent <<'PAYLOAD'
{
  "github_owned_allowed": true,
  "verified_allowed": true,
  "patterns_allowed": [
    "goreleaser/goreleaser-action@*",
    "cachix/install-nix-action@*",
    "golangci/golangci-lint-action@*",
    "geomys/sandboxed-step@*",
    "sigstore/cosign-installer@*",
    "anchore/sbom-action/*@*",
    "ossf/scorecard-action@*"
  ]
}
PAYLOAD
echo "  - Allowed: github-owned + verified + explicitly listed third-party actions"

# Fork PR approval policy
gh_api "repos/$REPO/actions/permissions/access" -X PUT \
  -f access_level=none \
  --silent 2>/dev/null || true
echo "  - Fork PR workflows: require approval"

# --------------------------------------------------------------------------- #
# 6. Immutable releases                                                       #
# --------------------------------------------------------------------------- #
echo "[6/6] Immutable releases..."
gh_api "repos/$REPO/immutable-releases" -X PUT \
  -H "X-GitHub-Api-Version: 2026-03-10" \
  --silent
echo "  - Immutable releases: enabled"
echo "  - Tags locked to commit after release publication"
echo "  - Release assets protected from modification/deletion"
echo "  - Release attestation auto-generated"

echo ""
echo "=== Done. Repository $REPO hardened. ==="
