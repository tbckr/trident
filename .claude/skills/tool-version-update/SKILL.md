---
name: tool-version-update
description: Use when the user references a `tool-update` GitHub issue from the trident `tool-versions.yml` workflow (e.g. "update tools from #16", "process tool-update issue 23", "do the pinned tool update", "aktualisiere die Tools aus Issue #16"). Updates Go tool versions (govulncheck, go-licenses, golangci-lint, goreleaser) pinned in .github/workflows/ and .goreleaser.yaml, verifies repo-wide consistency, and emits per-tool commit commands.
---

# tool-version-update

## Overview

This skill processes a GitHub Issue created by the trident `tool-versions.yml` workflow. Given an issue ID, it reads the real update data from `scripts/check-tool-versions.sh` (not the issue body), patches every pinned occurrence of each outdated tool version across the repo, verifies consistency, and outputs ready-to-run `git add`/`git commit` commands — one per tool, following the project's Conventional Commits style.

**The issue is only a trigger. The local script is the source of truth.**

## When to Use

**Use when:**
- User says "update tools from #N", "process tool-update issue N", or equivalent
- User shows a `tool-update`-labeled issue from this repo
- Working tree is clean (no uncommitted changes)

**Do NOT use when:**
- The issue has a different label or title — it's not from this workflow
- Working tree has uncommitted changes — stop and ask the user to stash/commit first

## Prerequisites

Verify before starting:
```bash
git diff --quiet && git diff --cached --quiet  # must succeed — no pending edits
gh auth status                                  # must be authenticated
command -v jq                                   # must be available
```

Must be run from the repo root: `/home/tbecker/git/trident`.

## Workflow

### Step 1 — Load and validate the issue

```bash
gh issue view <id> --json number,title,body,labels,state
```

**Hard stops** (abort with clear message if any fail):
- State must be `OPEN`
- Must have label `tool-update`
- Title must be exactly: `chore(deps): Go tool version updates available`

### Step 2 — Run the script as source of truth

```bash
REPORT_FILE=/tmp/trident-tool-updates.json ./scripts/check-tool-versions.sh || true
# exit 1 = updates available (expected); exit 0 = nothing to update
```

Parse the JSON report:
```bash
jq -c '.[]' /tmp/trident-tool-updates.json
# Each entry: {"name":"goreleaser","current":"v2.15.1","latest":"v2.15.2","source":"goreleaser/goreleaser","files":"..."}
```

**If the script exits 0** (no updates): The issue is stale. Stop and tell the user:
> "All tools are already up to date. Close issue #N manually."

**If the script report differs from the issue body** (e.g. issue mentions only goreleaser but script also finds golangci-lint): Use the script's report. Inform the user of the discrepancy.

### Step 3 — Update each tool

For each entry in the JSON report, apply the correct edit pattern.

#### Pattern A — `go run` tools: `govulncheck`, `go-licenses`

The pinned string is `<module-path>@v<version>` inside workflow YAML and `.goreleaser.yaml`.

Find all files containing it (use the `source` field from the JSON as the module path):
```bash
grep -rl "<source>@v<current>" .github/workflows/ .goreleaser.yaml
```

For each file found: replace `<source>@v<current>` with `<source>@v<latest>` using the Edit tool (exact string match).

#### Pattern B — Action input tools: `golangci-lint`, `goreleaser`

The pinned string is a `version:` field inside a `with:` block in a workflow YAML.

Find all files containing the version:
```bash
grep -rl "version: \"v<current>\"" .github/workflows/
# or without quotes:
grep -rl "version: v<current>" .github/workflows/
```

For each file found: replace only `version: "v<current>"` → `version: "v<latest>"` (or without quotes, matching the existing style).

**CRITICAL safety rules for Pattern B:**
- **NEVER change the action SHA** (`uses: goreleaser/goreleaser-action@<sha>  # v7.0.0`). The SHA is managed by Dependabot.
- **NEVER change the `# v7.0.0` comment** on the `uses:` line — that refers to the action version, not the tool version.
- Only touch the `version:` input line inside the `with:` block.

### Step 4 — Repo-wide safety net

After all edits, grep for any remaining occurrences of the old version strings:

```bash
# For each updated tool, check for stragglers
grep -rn "v<old>" .github/workflows/ .goreleaser.yaml
```

Ignore any hits inside `justfile` where the tool appears with `@latest` — those are intentionally unpinned (lines 41/45 in justfile).

Any other remaining hits: **warn the user** but do not abort — let them decide.

### Step 5 — Verify

Re-run the script. It must exit 0 for the tools that were just updated:

```bash
REPORT_FILE=/tmp/trident-verify.json ./scripts/check-tool-versions.sh
echo "Exit: $?"
```

If it still exits 1 and lists the same tools: the edit did not take effect — show the diff and stop.

After verification passes, suggest (but do not run):
```
Consider running: just lint
Full CI check:    just ci
```

### Step 6 — Output commit commands

Print one `git add` + `git commit` block per updated tool. Use the Conventional Commits format the project already uses:

```
## Suggested commits (one per tool)

# goreleaser: v2.15.1 → v2.15.2
git add .github/workflows/release.yml .github/workflows/goreleaser-lint.yml
git commit -m "chore(deps): bump goreleaser from v2.15.1 to v2.15.2" -m "Closes #<issue-id>"

# govulncheck: v1.1.4 → v1.1.5  (if multiple tools, only last commit needs Closes)
git add .github/workflows/ci.yml .github/workflows/vuln-schedule.yml .goreleaser.yaml
git commit -m "chore(deps): bump govulncheck from v1.1.4 to v1.1.5"
```

List exactly the files that were changed for each tool. No more, no less.

## Commands Reference

| Purpose | Command |
|---|---|
| Load issue | `gh issue view <id> --json number,title,body,labels,state` |
| Run script | `REPORT_FILE=/tmp/x.json ./scripts/check-tool-versions.sh \|\| true` |
| Parse report | `jq -c '.[]' /tmp/x.json` |
| Find go-run occurrences | `grep -rl "<module>@v<old>" .github/workflows/ .goreleaser.yaml` |
| Find action-input occurrences | `grep -rl "version: \"v<old>\"" .github/workflows/` |
| Safety net check | `grep -rn "v<old>" .github/workflows/ .goreleaser.yaml` |
| Verify | `REPORT_FILE=/tmp/v.json ./scripts/check-tool-versions.sh; echo $?` |

## Tool Registry (for reference)

| Tool | Type | Module/Repo | Pinned in |
|---|---|---|---|
| `govulncheck` | go run | `golang.org/x/vuln/cmd/govulncheck` | `ci.yml`, `vuln-schedule.yml`, `.goreleaser.yaml` |
| `go-licenses` | go run | `github.com/google/go-licenses/v2` | `ci.yml` |
| `golangci-lint` | action input | `golangci/golangci-lint` | `ci.yml` |
| `goreleaser` | action input | `goreleaser/goreleaser` | `release.yml`, `goreleaser-lint.yml` |

## Common Mistakes

| Mistake | Consequence | Prevention |
|---|---|---|
| Changing action SHA (`@abc123…`) | Breaks SHA pinning, Dependabot conflict | Only change `version:` line, never the `uses:` line |
| Changing `justfile` `@latest` lines | Breaks intentional unpinned usage | Ignore justfile entirely |
| Only updating `release.yml` for goreleaser | `goreleaser-lint.yml` left on old version | Always run `grep -rl` — don't rely on script's file list alone |
| Parsing issue body instead of running script | Acts on stale data | Always run script first, use JSON report |
| Skipping Step 5 verification | Ship without confirming the fix | Never skip — the script is fast |
| Two `%w` in one `fmt.Errorf` | (Go-specific, not applicable here) | N/A |
