#!/usr/bin/env bash
set -euo pipefail

# generate-provenance.sh — generate a SLSA Provenance v1 predicate JSON
#
# Usage:
#   ./scripts/generate-provenance.sh [ARTIFACTS_JSON]
#
# Default ARTIFACTS_JSON: dist/artifacts.json
#
# Inputs (environment variables, all available on GitHub Actions):
#   GITHUB_REPOSITORY   — owner/repo (e.g. tbckr/trident)
#   GITHUB_SHA          — full commit hash
#   GITHUB_REF          — refs/tags/vX.Y.Z
#   GITHUB_RUN_ID       — unique run ID
#   GITHUB_RUN_ATTEMPT  — attempt number
#   GITHUB_SERVER_URL   — https://github.com
#   GITHUB_WORKFLOW_REF — workflow file path + ref (e.g. owner/repo/.github/workflows/release.yml@refs/tags/v1.0.0)
#   RUNNER_OS           — Linux
#   RUNNER_ARCH         — X64
#   BUILD_STARTED_ON    — RFC3339 timestamp set before GoReleaser runs (written to GITHUB_ENV)
#
# Output: dist/provenance-predicate.json
#
# Requirements:
#   - jq

ARTIFACTS_JSON="${1:-dist/artifacts.json}"
OUTPUT="dist/provenance-predicate.json"

# Validate required environment variables
required_vars=(
  GITHUB_REPOSITORY
  GITHUB_SHA
  GITHUB_REF
  GITHUB_RUN_ID
  GITHUB_RUN_ATTEMPT
  GITHUB_SERVER_URL
  GITHUB_WORKFLOW_REF
  RUNNER_OS
  RUNNER_ARCH
  BUILD_STARTED_ON
)

for var in "${required_vars[@]}"; do
  if [[ -z "${!var:-}" ]]; then
    echo "Error: required environment variable $var is not set" >&2
    exit 1
  fi
done

if [[ ! -f "$ARTIFACTS_JSON" ]]; then
  echo "Error: artifacts file not found: $ARTIFACTS_JSON" >&2
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo "Error: jq is required but not installed" >&2
  exit 1
fi

# Extract workflow filename from GITHUB_WORKFLOW_REF
# Format: owner/repo/.github/workflows/file.yml@refs/tags/v1.0.0
WORKFLOW_FILE=$(echo "$GITHUB_WORKFLOW_REF" | sed 's|.*/.github/workflows/||' | sed 's|@.*||')

STARTED_ON="$BUILD_STARTED_ON"

# Build byproducts array from SBOM entries in artifacts.json.
# Each SBOM entry has: name, extra.Checksum (format: "sha256:<hex>")
BYPRODUCTS=$(jq -c '[
  .[] |
  select(.type == "SBOM") |
  {
    name: .name,
    digest: {
      sha256: (.extra.Checksum | ltrimstr("sha256:"))
    },
    mediaType: "application/vnd.cyclonedx+json"
  }
]' "$ARTIFACTS_JSON")

FINISHED_ON=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Generate the SLSA Provenance v1 predicate
jq -n \
  --arg repository "${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}" \
  --arg ref "$GITHUB_REF" \
  --arg workflow ".github/workflows/${WORKFLOW_FILE}" \
  --arg runnerOS "$RUNNER_OS" \
  --arg runnerArch "$RUNNER_ARCH" \
  --arg commit "$GITHUB_SHA" \
  --arg runId "$GITHUB_RUN_ID" \
  --arg runAttempt "$GITHUB_RUN_ATTEMPT" \
  --arg invocationId "${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}/attempts/${GITHUB_RUN_ATTEMPT}" \
  --arg startedOn "$STARTED_ON" \
  --arg finishedOn "$FINISHED_ON" \
  --argjson byproducts "$BYPRODUCTS" \
  '{
    buildDefinition: {
      buildType: "https://github.com/tbckr/trident/build/goreleaser@v1",
      externalParameters: {
        repository: $repository,
        ref: $ref,
        workflow: $workflow
      },
      internalParameters: {
        runnerOS: $runnerOS,
        runnerArch: $runnerArch,
        goreleaserConfig: ".goreleaser.yaml"
      },
      resolvedDependencies: [
        {
          uri: ("git+" + $repository + "@" + $ref),
          digest: {
            gitCommit: $commit
          }
        }
      ]
    },
    runDetails: {
      builder: {
        id: "https://github.com/actions/runner"
      },
      metadata: {
        invocationId: $invocationId,
        startedOn: $startedOn,
        finishedOn: $finishedOn
      },
      byproducts: $byproducts
    }
  }' > "$OUTPUT"

echo "==> Provenance predicate written to $OUTPUT"
