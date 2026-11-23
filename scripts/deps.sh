#!/usr/bin/env bash
# Dependency maintenance (CONTRIBUTING.md: Updating dependencies).
# latest-deps.yml runs the upgrade weekly in a sandbox and tests the result
# without committing it, checks the Go release in go.mod and puts the list of
# `outdated` into its summary for review.
#
# Usage: scripts/deps.sh <command>
#   outdated    what the upgrade leaves out, for review; never fails: direct
#               modules with a newer release or a deprecation, the Go release
#               in go.mod (and that .go-version and flake.nix match it), the
#               tools and container images pinned by hand, the flake inputs
#               and the notes in code and configuration that say Revisit or
#               Drop this once
#   upgrade     every direct module to its newest release, then go mod tidy
#   go-support  fail if the Go release in go.mod no longer gets security fixes
#
# Only direct dependencies are upgraded: `go get <module>@latest` for each
# direct module. Minimal version selection raises an indirect module only to
# the minimum the new version requires; `go get -u` would raise indirect
# modules on its own. Major versions (a new module path), the Go release, the
# pinned tools and images and the flake inputs stay manual steps.
#
# `outdated` reads the pins in the forms the workflows write them, so keep
# these forms when bumping:
#   golangci-lint  GOLANGCI_LINT_VERSION: vX.Y.Z   ci.yml
#   GoReleaser     GORELEASER_VERSION: vX.Y.Z      release.yml, goreleaser-lint.yml
#   govulncheck    golang.org/x/vuln/cmd/govulncheck@vX.Y.Z
#   go-licenses    github.com/google/go-licenses/v2@vX.Y.Z
#                  (both in any workflow and in .goreleaser.yaml)
#   semgrep        image: semgrep/semgrep:X.Y.Z@sha256:<digest>      semgrep.yml
#   betterleaks    image: ghcr.io/betterleaks/betterleaks:vX.Y.Z@sha256:<digest>
#                                                                   betterleaks.yml
#
# Environment:
#   GITHUB_TOKEN  optional; raises the GitHub API rate limit of `outdated`
set -euo pipefail

self="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)/$(basename -- "$0")"
CDPATH='' cd -- "$(dirname -- "$self")/.."

workflows=.github/workflows

usage() {
  sed -n '/^# Usage/,/^#$/{/^#$/d;s/^# \{0,1\}//;p}' "$self" >&2
  exit 2
}

upgrade() {
  local modules
  modules="$(go list -m -f '{{if not (or .Indirect .Main)}}{{.Path}}{{end}}' all |
    awk 'NF { print $0 "@latest" }')"
  if [[ -n "$modules" ]]; then
    # shellcheck disable=SC2086 # one module per word
    go get $modules
  fi
  go mod tidy
}

# The Go release in go.mod, such as 1.26.
go_release() {
  go list -m -f '{{.GoVersion}}' | cut -d. -f1,2
}

# The Go releases that get security fixes, oldest first: the Go project
# supports the two most recent major releases. The list of releases comes from
# the golang.org/toolchain module on the module proxy.
supported_go() {
  go list -m -versions -f '{{range .Versions}}{{println .}}{{end}}' golang.org/toolchain |
    sed -nE 's/^v0\.0\.1-go(1\.[0-9]+)\.[0-9]+\.linux-amd64$/\1/p' |
    sort -uV | tail -n 2 | paste -sd ' ' -
}

go_support() {
  local current supported
  current="$(go_release)"
  supported="$(supported_go)"
  if [[ -z "$supported" ]]; then
    echo "deps: no Go releases in golang.org/toolchain" >&2
    return 1
  fi
  if [[ " $supported " != *" $current "* ]]; then
    echo "Go $current (go.mod) no longer gets security fixes; supported: $supported." >&2
    echo "Raise the go directive in go.mod, and with it the release in .go-version and buildGo1XXModule and go_1_XX in flake.nix." >&2
    return 1
  fi
  echo "Go $current (go.mod) is supported; supported: $supported."
}

# True if the second version is a later one than the first.
newer() {
  [[ -n "$2" && "$1" != "$2" && "$(printf '%s\n' "$1" "$2" | sort -V | tail -n 1)" == "$2" ]]
}

# One line of the report: "label current -> newest (where)" if a later
# version exists.
report() {
  local label=$1 current=$2 newest=$3 where=$4
  if [[ -z "$newest" ]]; then
    echo "$label $current ($where): latest unknown"
  elif newer "$current" "$newest"; then
    echo "$label $current -> $newest ($where)"
  else
    echo "$label $current ($where): up to date"
  fi
}

# The versions that a pattern pins in the files, one "version file" per line
# in the order found. The pattern is an extended regular expression without
# "|" whose first group is the version.
pins() {
  local pattern=$1 file
  shift
  for file in "$@"; do
    sed -nE "s|.*$pattern.*|\\1 ${file##*/}|p" "$file"
  done | awk '!seen[$0]++'
}

# The files of the pins (lines of `pins`) that pin a version, comma-separated.
files_of() {
  awk -v version="$2" '$1 == version { printf "%s%s", sep, $2; sep = ", " } END { print "" }' <<<"$1"
}

# The report line of a tool pinned in one or more files. If the pins
# disagree, it lists each version with its files instead.
pinned() {
  local label=$1 newest=$2 found=$3 versions version where=""
  if [[ -z "$found" ]]; then
    echo "$label: no pin found"
    return
  fi
  versions="$(cut -d' ' -f1 <<<"$found" | sort -uV)"
  if [[ "$versions" != *$'\n'* ]]; then
    report "$label" "$versions" "$newest" "$(files_of "$found" "$versions")"
    return
  fi
  while read -r version; do
    where+="${where:+; }$version in $(files_of "$found" "$version")"
  done <<<"$versions"
  echo "$label pins disagree ($where); latest ${newest:-unknown}"
}

# The tag and digest of an image pin in a file, "tag sha256:<digest>".
image_pin() {
  local file=$1 image=$2
  sed -nE "s|.*image: *${image//./\\.}:([^@[:space:]]+)@(sha256:[0-9a-f]+).*|\\1 \\2|p" "$file" | head -n 1
}

# The report line of an image pinned by tag and digest. A newer tag comes
# with the pin to use; so does the pinned tag if the registry now serves
# another digest for it, because it was rebuilt.
image_report() {
  local label=$1 image=$2 pin=$3 newest=$4 newest_digest=$5 where=$6 current digest
  if [[ -z "$pin" ]]; then
    echo "$label: no pin found ($where)"
    return
  fi
  current="${pin% *}"
  digest="${pin#* }"
  if [[ -n "$newest" ]] && newer "$current" "$newest"; then
    echo "$label $current -> $newest ($where), pin $image:$newest@${newest_digest:-<digest unknown>}"
  elif [[ "$newest" == "$current" && -z "$newest_digest" ]]; then
    echo "$label $current ($where): up to date, digest unknown"
  elif [[ "$newest" == "$current" && "$newest_digest" != "$digest" ]]; then
    echo "$label $current ($where): rebuilt, pin $image:$current@$newest_digest"
  else
    report "$label" "$current" "$newest" "$where"
  fi
}

# The newest release of a module on the module proxy, within its major.
module_latest() {
  go list -m -f '{{.Version}}' "$1@latest"
}

# The tag of the latest release of a GitHub repository, any major.
github_latest() {
  local auth=()
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    auth=(-H "Authorization: Bearer $GITHUB_TOKEN")
  fi
  curl -fsSL --max-time 30 ${auth[@]+"${auth[@]}"} -H 'Accept: application/vnd.github+json' \
    "https://api.github.com/repos/$1/releases/latest" | jq -r '.tag_name // empty'
}

# The newest X.Y.Z tag of a Docker Hub repository among its 100 most recently
# updated tags; pre-releases and rolling tags do not match.
dockerhub_newest() {
  curl -fsSL --max-time 30 "https://hub.docker.com/v2/repositories/$1/tags/?page_size=100&ordering=last_updated" |
    jq -r '.results[].name | select(test("^[0-9]+\\.[0-9]+\\.[0-9]+$"))' | sort -V | tail -n 1
}

# The digest of a tag on Docker Hub (of its manifest list, as pinned).
dockerhub_digest() {
  curl -fsSL --max-time 30 "https://hub.docker.com/v2/repositories/$1/tags/$2/" | jq -r '.digest // empty'
}

# The digest of a tag on ghcr.io (of its image index, as pinned), with an
# anonymous pull token.
ghcr_digest() {
  local token
  token="$(curl -fsSL --max-time 30 "https://ghcr.io/token?service=ghcr.io&scope=repository:$1:pull" |
    jq -r '.token // empty')"
  [[ -n "$token" ]] || return 1
  curl -fsSIL --max-time 30 -H "Authorization: Bearer $token" \
    -H 'Accept: application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json' \
    "https://ghcr.io/v2/$1/manifests/$2" |
    tr -d '\r' | awk 'tolower($1) == "docker-content-digest:" { print $2 }' | tail -n 1
}

outdated() {
  local found current supported newest newest_digest pin flake goversion

  current="$(go_release)" || current=""
  supported="$(supported_go)" || supported=""

  echo "== Go: direct modules with a newer release or a deprecation"
  # go list -u leaves out a module whose lookup failed, without an error.
  # Without an answer from the module proxy for the Go releases, the list
  # would say none. A module that GOPRIVATE or GONOPROXY sends past the proxy
  # can still go missing.
  if [[ -z "$supported" ]]; then
    echo "lookup failed: no answer from the module proxy"
  elif found="$(go list -m -u -f '{{if and (not (or .Main .Indirect)) (or .Update .Deprecated)}}{{.Path}} {{.Version}}{{with .Update}} -> {{.Version}}{{end}}{{with .Deprecated}} (deprecated: {{.}}){{end}}{{end}}' all)"; then
    found="$(awk NF <<<"$found")"
    echo "${found:-none}"
  else
    echo "lookup failed"
  fi

  echo
  echo "== Platform"
  if [[ -z "$supported" ]]; then
    echo "Go $current (go.mod): supported releases unknown"
  elif [[ " $supported " != *" $current "* ]]; then
    echo "Go $current (go.mod): no longer supported; supported: $supported"
  elif newer "$current" "${supported##* }"; then
    echo "Go $current (go.mod): supported, newest ${supported##* }"
  else
    echo "Go $current (go.mod): up to date"
  fi
  flake="$(sed -nE 's/.*buildGo1([0-9]+)Module.*/1.\1/p; s/.*go_1_([0-9]+).*/1.\1/p' flake.nix | sort -u | paste -sd ' ' -)" ||
    flake=""
  if [[ "$flake" == "$current" ]]; then
    echo "Go $flake (flake.nix: buildGo1XXModule, go_1_XX): matches go.mod"
  else
    echo "Go ${flake:-unknown} (flake.nix: buildGo1XXModule, go_1_XX): go.mod has $current"
  fi
  # setup-go reads .go-version, which holds the release only (X.Y), so that
  # check-latest installs its newest patch.
  goversion="$(tr -d '[:space:]' <.go-version)" || goversion=""
  if [[ "$goversion" == "$current" ]]; then
    echo "Go $goversion (.go-version: setup-go in the workflows): matches go.mod"
  else
    echo "Go ${goversion:-unknown} (.go-version: setup-go in the workflows): go.mod has $current"
  fi

  newest="$(github_latest golangci/golangci-lint)" || newest=""
  pinned golangci-lint "$newest" "$(pins 'GOLANGCI_LINT_VERSION: *"?(v[0-9.]+)' "$workflows/ci.yml")"
  newest="$(github_latest goreleaser/goreleaser)" || newest=""
  pinned GoReleaser "$newest" \
    "$(pins 'GORELEASER_VERSION: *"?(v[0-9.]+)' "$workflows/release.yml" "$workflows/goreleaser-lint.yml")"
  newest="$(module_latest golang.org/x/vuln)" || newest=""
  pinned govulncheck "$newest" \
    "$(pins 'golang\.org/x/vuln/cmd/govulncheck@(v[0-9.]+)' "$workflows"/*.yml .goreleaser.yaml)"
  newest="$(module_latest github.com/google/go-licenses/v2)" || newest=""
  pinned go-licenses "$newest" \
    "$(pins 'github\.com/google/go-licenses/v2@(v[0-9.]+)' "$workflows"/*.yml .goreleaser.yaml)"

  pin="$(image_pin "$workflows/semgrep.yml" semgrep/semgrep)" || pin=""
  newest="$(dockerhub_newest semgrep/semgrep)" || newest=""
  newest_digest=""
  if [[ -n "$newest" ]]; then
    newest_digest="$(dockerhub_digest semgrep/semgrep "$newest")" || newest_digest=""
  fi
  image_report semgrep semgrep/semgrep "$pin" "$newest" "$newest_digest" semgrep.yml

  pin="$(image_pin "$workflows/betterleaks.yml" ghcr.io/betterleaks/betterleaks)" || pin=""
  newest="$(github_latest betterleaks/betterleaks)" || newest=""
  newest_digest=""
  if [[ -n "$newest" ]]; then
    newest_digest="$(ghcr_digest betterleaks/betterleaks "$newest")" || newest_digest=""
  fi
  image_report betterleaks ghcr.io/betterleaks/betterleaks "$pin" "$newest" "$newest_digest" betterleaks.yml

  jq -r '.nodes as $nodes | .nodes.root.inputs | to_entries[] | .key as $input | $nodes[.value]
    | "\($input) \(.original.ref // "(default branch)") locked \(.locked.lastModified | strftime("%Y-%m-%d")),"
      + " \((now - .locked.lastModified) / 86400 | floor) days ago (flake.lock)"' flake.lock ||
    echo "flake inputs unknown (flake.lock)"

  echo
  echo "== Reminders"
  git grep -n -E '(Revisit|Drop this once) ' -- ':!*.md' ':!scripts/deps.sh' |
    sed -E 's/^([^:]+:[0-9]+:)[[:space:]#/]*/\1 /' || echo "none"
}

case "${1:-}" in
  outdated) outdated ;;
  upgrade) upgrade ;;
  go-support) go_support ;;
  *) usage ;;
esac
