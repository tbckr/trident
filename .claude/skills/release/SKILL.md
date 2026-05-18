---
name: release
description: Guarded wrapper around `just release` for trident. Confirms tree state, classifies commits to predict the svu bump, flags `chore(deps):` commits that may need to be `fix(deps):` for the release to fire, runs `just ci`, then tags and pushes. Invoke explicitly via `/release [kind]` — model-invocation disabled because this pushes signed tags. `kind` defaults to `next`; accepts `next`/`major`/`minor`/`patch`/`prerelease`.
disable-model-invocation: true
---

# release

## Overview

Releases are driven by `svu` + signed git tags. The svu version bump derives from Conventional Commit prefixes since the last tag:

- `feat:` → minor bump
- `fix:` → patch bump
- `chore:` / `docs:` / `style:` alone → **no bump** (svu returns the current version → empty tag)

A historically recurring footgun: a security-driven dep bump committed as `chore(deps):` instead of `fix(deps):` looks identical to a routine chore — svu skips the bump, and the security fix never ships. This skill warns when the upcoming bump set looks suspicious.

This skill ONLY runs after the user explicitly says `/release` (with optional kind).

## When to Use

- User says `/release`, `/release patch`, `cut a release`, `tag the next version`
- Working on `main`, after merging the changes that should ship

Do not use to back-fill a tag. Do not use for `chore:`-only branches.

## Prerequisites

```bash
# Must succeed before any tagging:
test -f justfile                                   # repo root
command -v svu                                     # tagging engine
command -v gh                                      # workflow watcher
git diff --quiet && git diff --cached --quiet     # tree clean (HARD STOP if dirty)
[ "$(git rev-parse --abbrev-ref HEAD)" = "main" ]  # on main
git fetch origin main --tags
[ "$(git rev-list --count HEAD..origin/main)" = "0" ] # up to date with origin
```

If any of these fail: explain the failure to the user and stop.

## Workflow

### Step 1 — Show the current state

```bash
git describe --tags --abbrev=0      # last tag
svu current                          # what svu thinks current is
```

### Step 2 — Classify commits since the last tag

Include the body so the `chore(deps):` security-context heuristic in this step can actually fire:

```bash
git log "$(svu current)..HEAD" --pretty='format:%h %s%n%b%n---' --no-merges
```

Walk the output and bucket each commit by Conventional Commit prefix on its subject:

| Prefix | Effect on svu next |
|--------|--------------------|
| `feat(...):` or `feat:` | minor bump |
| `fix(...):` or `fix:` | patch bump |
| `feat!:` / `fix!:` / `BREAKING CHANGE:` in body | major bump |
| `chore:` / `docs:` / `style:` / `refactor:` / `test:` | **no bump** |

**Special warning for `chore(deps):`**: scan each `chore(deps):` subject and its body (the `%b` you added above) for any of: `CVE-`, `security`, `vulnerab`, `govulncheck`, `GHSA-`. If a match appears, flag it loudly:

> `<sha> chore(deps): bump <pkg>` — looks security-driven (matched: `<keyword>` in body). Per project convention, govulncheck/CVE bumps should use `fix(deps):` so svu actually bumps the patch version. Consider amending the commit or adding a follow-up `fix(deps):` commit before releasing.

If a `chore(deps):` matches no security keyword and the body looks like a routine Dependabot/transitive bump — don't flag it.

### Step 3 — Show the predicted next version

```bash
echo "current: $(svu current)"
echo "next:    $(svu next)"
echo "patch:   $(svu patch)"
echo "minor:   $(svu minor)"
echo "major:   $(svu major)"
```

If `svu next == svu current`: **stop**. No version bump → no release. Report this and the bucketed commits to the user.

If `svu next` matches the user's expectation: continue.

### Step 4 — Confirm with the user

Print a release plan (use `just verify-release` later):

```
Release plan:
  current tag: <svu current>
  next tag:    <svu next>     (kind: <kind>)
  branch:      main
  commits in this tag: <count>
  notable: <list of feat/fix subjects, one per line>

Run `just ci` first, then `just release <kind>`?
```

**Take affirmative consent before proceeding.** The next steps tag, sign, and push — at that point the `release.yml` workflow attestates artifacts and consumers can fetch within seconds. There's no clean rollback (force-deleting a published tag breaks attestation chains and surprises anyone who already fetched it). If the user's response is ambiguous ("ok", "looks fine", "I guess"), read the release plan back and ask once more for a clearer signal. A short "yes", "go", "ship it" is enough — the goal isn't ritual, it's making sure the user has actually looked.

### Step 5 — Run CI locally

```bash
just ci
```

If anything fails: stop. Do not retry, do not skip, do not `--no-verify`. Hand the failure to the user.

### Step 6 — Run the release target

```bash
just release <kind>     # default "next"
```

This will: compute the version via `svu <kind>`, create a signed tag (`git tag -s`), then `git push && git push --tags`. The `release.yml` workflow fires on the tag push.

### Step 7 — Watch the release workflow

The workflow run may take a few seconds to register after `git push --tags`. Find the run by filtering on the tagged commit so you don't accidentally pick up an earlier push run:

```bash
sha=$(git rev-parse HEAD)
run_id=""
for _ in 1 2 3 4 5 6; do
  run_id=$(gh run list --workflow=release.yml --event=push --limit=10 \
    --json databaseId,headSha \
    --jq ".[] | select(.headSha == \"$sha\") | .databaseId" | head -1)
  [ -n "$run_id" ] && break
  sleep 5
done
[ -z "$run_id" ] && { echo "release.yml run not found for $sha — check Actions tab"; exit 1; }
gh run watch --exit-status "$run_id"
```

If the workflow fails: tell the user the run URL and stop. Do not delete the tag — published tags should stay published (consumers may already have fetched; force-deleting breaks attestation).

### Step 8 — Verify one release artifact

After Step 7 exits 0, the workflow has produced and attestated the assets. Pick one platform archive from the release (e.g. linux x86_64), download it, and run the existing verification script:

```bash
# List artifacts
gh release view "$(svu current)" --json assets --jq '.assets[].name'

# Pick one tarball, download to /tmp, then:
just verify-release /tmp/<archive-name>
```

`scripts/verify-release.sh` runs `gh attestation verify` and reports success or the specific failure.

## Commands Reference

| Purpose | Command |
|---|---|
| Last tag | `git describe --tags --abbrev=0` |
| svu current/next | `svu current` / `svu next` |
| Commits since last tag | `git log "$(svu current)..HEAD" --pretty='format:%h %s' --no-merges` |
| Run CI | `just ci` |
| Tag + push | `just release <kind>` |
| Watch workflow | `gh run watch --exit-status <run-id>` |
| List release assets | `gh release view <tag> --json assets --jq '.assets[].name'` |
| Verify artifact | `just verify-release <archive>` |

## Hard stops (refuse to proceed)

| Condition | Reason |
|-----------|--------|
| Dirty working tree | Tag would point at an inconsistent state |
| Not on `main` | Releases only fire from `main` |
| Behind `origin/main` | Tag would skip merged commits |
| `svu next == svu current` | No version bump → no tag → no release |
| `just ci` failed | Ship green or don't ship |
| `git tag -s` failed (GPG) | Check `gpg --card-status` / agent / unlocked passphrase. Never fall back to an unsigned tag — release attestation expects a signed tag. |
| Affirmative consent not given | Tagging + push is irreversible (see Step 4 reasoning) |

## Common Mistakes

| Mistake | Consequence | Prevention |
|---------|-------------|------------|
| Treating `chore(deps):` security bumps as routine | svu skips the bump → fix never released | Step 2 flag |
| Skipping `just ci` because "CI will catch it" | Tag exists with a broken release workflow | Step 5 is mandatory |
| Confirming with "ok" | Premature tag push | Step 4 requires explicit yes |
| Releasing from a feature branch | Wrong tag target | Step 1 hard stop |
| Force-pushing or deleting a published tag | Breaks consumers, breaks attestation chain | Never. Discuss with user first |
