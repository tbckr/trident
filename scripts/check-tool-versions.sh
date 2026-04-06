#!/usr/bin/env bash
set -euo pipefail

# Tool version checker for pinned Go tools not covered by Dependabot.
# Parses current versions directly from workflow files — no dual maintenance.
#
# Registry format: NAME|TYPE|SOURCE|GREP_PATTERN|FILES
#   TYPE: "goproxy" (Go module proxy) or "github" (GitHub releases)
#   SOURCE: module path (goproxy) or owner/repo (github)
#   GREP_PATTERN: regex to extract current version from workflow files
#   FILES: file or directory to search in
#
# Environment:
#   GITHUB_TOKEN  — optional; avoids GitHub API rate limits
#   REPORT_FILE   — optional; write JSON report for CI consumption

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORKFLOWS="${REPO_ROOT}/.github/workflows"

TOOLS=(
  "govulncheck|goproxy|golang.org/x/vuln|govulncheck@(v[0-9.]+)|${WORKFLOWS}/ ${REPO_ROOT}/.goreleaser.yaml"
  "go-licenses|goproxy|github.com/google/go-licenses/v2|go-licenses/v2@(v[0-9.]+)|${WORKFLOWS}/"
  "golangci-lint|github|golangci/golangci-lint|golangci-lint-action|${WORKFLOWS}/ci.yml"
  "goreleaser|github|goreleaser/goreleaser|goreleaser-action|${WORKFLOWS}/release.yml ${WORKFLOWS}/goreleaser-lint.yml"
)

REPORT_FILE="${REPORT_FILE:-}"

# Extract current version from workflow files.
# For go run tools: parses "tool@vX.Y.Z" directly.
# For action version inputs: finds the action, then reads the next "version:" line.
extract_version() {
  local name="$1" pattern="$2" files="$3"
  case "$name" in
    golangci-lint|goreleaser)
      # Action with "version:" input — grep for the action, then the version field
      grep -A10 -r "$pattern" "$files" \
        | grep -oP 'version:\s*"?\Kv[0-9.]+' \
        | head -1
      ;;
    *)
      # "go run module@vX.Y.Z" pattern
      grep -rhoP "${pattern}" $files \
        | head -1 \
        | grep -oP 'v[0-9.]+$'
      ;;
  esac
}

check_go_proxy() {
  local module="$1"
  local encoded
  encoded=$(printf '%s' "$module" | sed 's|[A-Z]|!&|g' | tr '[:upper:]' '[:lower:]')
  curl -sfL "https://proxy.golang.org/${encoded}/@latest" | jq -r '.Version // empty' 2>/dev/null || true
}

check_github_release() {
  local repo="$1"
  local auth_header=()
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    auth_header=(-H "Authorization: token ${GITHUB_TOKEN}")
  fi
  curl -sfL "${auth_header[@]}" "https://api.github.com/repos/${repo}/releases/latest" \
    | jq -r '.tag_name // empty' 2>/dev/null || true
}

# Find all workflow files containing the pinned version for a tool.
find_locations() {
  local name="$1" pattern="$2" files="$3" current="$4"
  case "$name" in
    golangci-lint|goreleaser)
      grep -rl "$pattern" $files 2>/dev/null | sed "s|${REPO_ROOT}/||g" | paste -sd ',' -
      ;;
    *)
      grep -rl "${current}" $files 2>/dev/null | sed "s|${REPO_ROOT}/||g" | paste -sd ',' -
      ;;
  esac
}

has_updates=false
updates_json="[]"

printf "%-20s %-15s %-15s %s\n" "TOOL" "CURRENT" "LATEST" "STATUS"
printf "%-20s %-15s %-15s %s\n" "----" "-------" "------" "------"

for entry in "${TOOLS[@]}"; do
  IFS='|' read -r name type source pattern files <<< "$entry"

  current=$(extract_version "$name" "$pattern" "$files" 2>/dev/null || true)
  if [[ -z "$current" ]]; then
    printf "%-20s %-15s %-15s %s\n" "$name" "NOT FOUND" "-" "could not parse"
    continue
  fi

  case "$type" in
    goproxy) latest=$(check_go_proxy "$source") ;;
    github)  latest=$(check_github_release "$source") ;;
    *)       echo "Unknown type: $type for $name" >&2; continue ;;
  esac

  if [[ -z "$latest" ]]; then
    printf "%-20s %-15s %-15s %s\n" "$name" "$current" "ERROR" "could not fetch"
    continue
  fi

  if [[ "$current" == "$latest" ]]; then
    printf "%-20s %-15s %-15s %s\n" "$name" "$current" "$latest" "up-to-date"
  else
    locations=$(find_locations "$name" "$pattern" "$files" "$current")
    printf "%-20s %-15s %-15s %s\n" "$name" "$current" "$latest" "UPDATE AVAILABLE"
    has_updates=true
    updates_json=$(printf '%s' "$updates_json" | jq -c \
      --arg name "$name" \
      --arg current "$current" \
      --arg latest "$latest" \
      --arg source "$source" \
      --arg files "$locations" \
      '. + [{"name": $name, "current": $current, "latest": $latest, "source": $source, "files": $files}]')
  fi
done

if [[ -n "$REPORT_FILE" ]]; then
  printf '%s' "$updates_json" > "$REPORT_FILE"
fi

if [[ "$has_updates" == "true" ]]; then
  echo ""
  echo "Tool updates are available."
  exit 1
else
  echo ""
  echo "All tools are up-to-date."
  exit 0
fi
