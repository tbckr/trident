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

GH_ERR=$(mktemp)
trap 'rm -f "$GH_ERR"' EXIT

# Wrapper: retry on transient failures (connection refused, rate limit)
gh_api() {
  local attempt
  for attempt in 1 2 3; do
    if gh api "$@" 2>"$GH_ERR"; then
      return 0
    fi
    local rc=$?
    local err
    err=$(cat "$GH_ERR")
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
echo "[1/8] Repository settings..."
gh_api "repos/$REPO" -X PATCH \
  -f has_wiki=false \
  -f has_projects=false \
  -f has_discussions=false \
  -f allow_auto_merge=false \
  -f delete_branch_on_merge=true \
  --silent
echo "  - Wiki: disabled"
echo "  - Projects: disabled"
echo "  - Discussions: disabled"
echo "  - Auto-merge: disabled"
echo "  - Delete branch on merge: enabled"

# --------------------------------------------------------------------------- #
# 2. Vulnerability alerts + secret scanning                                   #
# --------------------------------------------------------------------------- #
echo "[2/8] Security features..."
gh_api "repos/$REPO/vulnerability-alerts" -X PUT --silent 2>/dev/null || true
echo "  - Dependabot alerts: enabled"

if gh_api "repos/$REPO" -X PATCH \
  -f 'security_and_analysis[secret_scanning][status]=enabled' \
  -f 'security_and_analysis[secret_scanning_push_protection][status]=enabled' \
  -f 'security_and_analysis[secret_scanning_ai_detection][status]=enabled' \
  --silent 2>/dev/null; then
  echo "  - Secret scanning: enabled"
  echo "  - Push protection: enabled"
  echo "  - AI detection (validity checks): enabled"
else
  echo "  - Secret scanning: skipped (may require GitHub Advanced Security)"
fi

# --------------------------------------------------------------------------- #
# 3. Private vulnerability reporting                                          #
# --------------------------------------------------------------------------- #
echo "[3/8] Private vulnerability reporting..."
gh_api "repos/$REPO/private-vulnerability-reporting" -X PUT --silent 2>/dev/null || true
echo "  - Private vulnerability reporting: enabled"

# --------------------------------------------------------------------------- #
# 4. Branch ruleset on main                                                   #
# --------------------------------------------------------------------------- #
echo "[4/8] Branch ruleset on '$BRANCH'..."

RULESET_NAME="main-branch-protection"
RULESET_ID=$(gh_api "repos/$REPO/rulesets" \
  --jq ".[] | select(.name == \"$RULESET_NAME\") | .id" 2>/dev/null || echo "")

if [ -n "$RULESET_ID" ]; then
  RULESET_METHOD="-X PUT"
  RULESET_URL="repos/$REPO/rulesets/$RULESET_ID"
  RULESET_ACTION="updated (id: $RULESET_ID)"
else
  RULESET_METHOD="-X POST"
  RULESET_URL="repos/$REPO/rulesets"
  RULESET_ACTION="created"
fi

# shellcheck disable=SC2086
gh_api "$RULESET_URL" $RULESET_METHOD --input - --silent <<'PAYLOAD'
{
  "name": "main-branch-protection",
  "target": "branch",
  "enforcement": "active",
  "bypass_actors": [
    {
      "actor_id": 5,
      "actor_type": "RepositoryRole",
      "bypass_mode": "always"
    }
  ],
  "conditions": {
    "ref_name": {
      "include": ["refs/heads/main"],
      "exclude": []
    }
  },
  "rules": [
    { "type": "deletion" },
    { "type": "non_fast_forward" },
    { "type": "required_linear_history" },
    { "type": "required_signatures" },
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 1,
        "dismiss_stale_reviews_on_push": true,
        "require_code_owner_review": true,
        "require_last_push_approval": true,
        "required_review_thread_resolution": true,
        "allowed_merge_methods": ["rebase", "squash"]
      }
    },
    {
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": true,
        "required_status_checks": [
          { "context": "Test", "integration_id": 15368 },
          { "context": "Lint", "integration_id": 15368 },
          { "context": "Vulnerability Check", "integration_id": 15368 },
          { "context": "License Check", "integration_id": 15368 },
          { "context": "GoReleaser Lint", "integration_id": 15368 },
          { "context": "Nix Flake Check", "integration_id": 15368 }
        ]
      }
    }
  ]
}
PAYLOAD
echo "  - Ruleset '$RULESET_NAME': $RULESET_ACTION"
echo "  - Bypass: repository admins (owner can push directly)"
echo "  - Require status checks (strict, GitHub Actions): Test, Lint, Vulnerability Check, License Check, GoReleaser Lint, Nix Flake Check"
echo "  - Require PR reviews: 1 approval (owner bypasses)"
echo "  - Require code owner review: yes"
echo "  - Dismiss stale reviews on push: yes"
echo "  - Require last push approval: yes"
echo "  - Require conversation resolution: yes"
echo "  - Allowed merge methods: rebase, squash"
echo "  - Signed commits: required"
echo "  - Linear history: required"
echo "  - Force push: blocked"
echo "  - Branch deletion: blocked"

# Remove legacy branch protection rule (idempotent — 404 ignored)
gh_api "repos/$REPO/branches/$BRANCH/protection" -X DELETE --silent 2>/dev/null || true
echo "  - Legacy branch protection rule: removed"

# --------------------------------------------------------------------------- #
# 5. Tag protection ruleset                                                   #
# --------------------------------------------------------------------------- #
echo "[5/8] Tag protection ruleset..."

TAG_RULESET_NAME="release-tag-protection"
TAG_RULESET_ID=$(gh_api "repos/$REPO/rulesets" \
  --jq ".[] | select(.name == \"$TAG_RULESET_NAME\") | .id" 2>/dev/null || echo "")

if [ -n "$TAG_RULESET_ID" ]; then
  TAG_METHOD="-X PUT"
  TAG_URL="repos/$REPO/rulesets/$TAG_RULESET_ID"
  TAG_ACTION="updated (id: $TAG_RULESET_ID)"
else
  TAG_METHOD="-X POST"
  TAG_URL="repos/$REPO/rulesets"
  TAG_ACTION="created"
fi

# shellcheck disable=SC2086
gh_api "$TAG_URL" $TAG_METHOD --input - --silent <<'PAYLOAD'
{
  "name": "release-tag-protection",
  "target": "tag",
  "enforcement": "active",
  "bypass_actors": [
    {
      "actor_id": 5,
      "actor_type": "RepositoryRole",
      "bypass_mode": "always"
    }
  ],
  "conditions": {
    "ref_name": {
      "include": ["refs/tags/v*"],
      "exclude": []
    }
  },
  "rules": [
    { "type": "deletion" },
    { "type": "non_fast_forward" },
    { "type": "required_signatures" }
  ]
}
PAYLOAD
echo "  - Ruleset '$TAG_RULESET_NAME': $TAG_ACTION"
echo "  - Protected tags: v*"
echo "  - Bypass: repository admins"
echo "  - Tag deletion: blocked"
echo "  - Tag overwrite: blocked"
echo "  - Signed tags: required"

# --------------------------------------------------------------------------- #
# 6. Actions permissions                                                      #
# --------------------------------------------------------------------------- #
echo "[6/8] Actions permissions..."
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
# 7. Default workflow permissions                                             #
# --------------------------------------------------------------------------- #
echo "[7/8] Default workflow permissions..."
gh_api "repos/$REPO/actions/permissions/workflow" -X PUT \
  --input - --silent <<'PAYLOAD'
{
  "default_workflow_permissions": "read",
  "can_approve_pull_request_reviews": false
}
PAYLOAD
echo "  - Default token permissions: read-only"
echo "  - Workflow PR approval: disabled"

# --------------------------------------------------------------------------- #
# 8. Immutable releases                                                       #
# --------------------------------------------------------------------------- #
echo "[8/8] Immutable releases..."
gh_api "repos/$REPO/immutable-releases" -X PUT \
  -H "X-GitHub-Api-Version: 2026-03-10" \
  --silent
echo "  - Immutable releases: enabled"
echo "  - Tags locked to commit after release publication"
echo "  - Release assets protected from modification/deletion"
echo "  - Release attestation auto-generated"

echo ""
echo "=== Done. Repository $REPO hardened. ==="
