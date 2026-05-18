---
name: add-service
description: Scaffold a new trident OSINT service. Walks the rigid 4-file package layout, CLI registration in `internal/cli/services.go` (`allServices`) and `internal/cli/<name>.go`, and the 5 README sections that need updates. Invoke explicitly via `/add-service <name>` — model-invocation disabled.
disable-model-invocation: true
---

# add-service

## Overview

Adding a new OSINT service in trident has 11 distinct touchpoints. Most service-related bugs in this repo have come from missing one (a forgotten `allServices()` entry, a missing README section, a `req.Response` nil deref, a `table.Header` mis-typed as returning an error). This skill scaffolds the package from templates and walks the registration checklist.

**The templates are correct skeletons — implement TODOs, don't restructure them.**

## When to Use

- User says "add a new service called X", "scaffold the FOO service", "lege den FOO-Service an"
- A new OSINT data source has been chosen and PAP level decided
- Working tree is clean enough that the new files are obviously the new work

Do not use to refactor an existing service. Do not use to rename a service.

## Prerequisites

```bash
# Run from /home/tbecker/git/trident
test -f go.mod && test -d internal/services       # in repo root
git diff --quiet && git diff --cached --quiet     # clean tree (warn, don't abort)
```

## Inputs to collect from the user

1. **Service name** (lowercase, single word, no dashes — must work as a Go package name). Examples: `cymru`, `crtsh`, `pgp`.
2. **PAP level**: `RED` / `AMBER` / `GREEN` / `WHITE`. Most data-fetching services are `AMBER` (queries leak metadata). DNS / pattern-matching is usually `GREEN`. Highly invasive or active scanning is `RED`.
3. **Shape**: pick one —
   - **AggregateService** (one input → multiple record types, supports bulk): like `dns`, `apex`. Uses `MultiResult`.
   - **Single-input** (one input → one Result): like `threatminer`, `pgp`. Uses `MultiResult` only if bulk output is meaningful.
   - **Identify-style** (typed slice input, custom `Run` signature, no `MultiResult`, inline PAP check): like `identify`. Use this only if the input isn't a single string — the templates won't fit directly; reference `internal/services/identify/` for the divergent pattern.
4. **HTTP-based?** If yes, the service.go template keeps the `*req.Client` field; if not (e.g. pure DNS using `*net.Resolver`), swap the field and constructor argument.

If the user only said "add service foo" — ask them for (2) and (3) before generating files.

## Workflow

### Step 1 — Create the package

```bash
mkdir -p internal/services/<name>
```

Copy each template, substituting `<name>` (lowercase package name) and `<PAP>` (one of `RED`/`AMBER`/`GREEN`/`WHITE`):

| Template | Destination | Notes |
|----------|-------------|-------|
| `templates/doc.go.tmpl` | `internal/services/<name>/doc.go` | Always — every package needs it |
| `templates/service.go.tmpl` | `internal/services/<name>/service.go` | Always |
| `templates/result.go.tmpl` | `internal/services/<name>/result.go` | Always |
| `templates/multi_result.go.tmpl` | `internal/services/<name>/multi_result.go` | Aggregate shape only — delete file if single-input |
| `templates/service_test.go.tmpl` | `internal/services/<name>/service_test.go` | Always |

Use the `Read` tool to load each template, then `Write` it out to the destination with the substitutions applied. After writing, scan for leftover `<name>` / `<PAP>` placeholders — fail loudly if any remain. For identify-style services, skip `multi_result.go.tmpl` and adapt `service.go.tmpl`'s `Run` signature to take the typed slice (see `internal/services/identify/service.go`).

### Step 2 — Fill in TODOs

The templates have explicit `// TODO:` comments at every decision point. Walk them in this order:

1. `service.go`: PAP constant, `Run` body (input validation → fetch → parse → return).
2. `result.go`: fields on `Result`, `IsEmpty` predicate, `WriteText` / `WriteTable` output.
3. `multi_result.go` (if kept): merged-table column layout.
4. `service_test.go`: at least one happy-path test + one error test + one empty-result test.

### Step 3 — Register in CLI

Edit `internal/cli/services.go` — add to the `metas` slice (`services.go:39-51`), alphabetical within group:

```go
{<name>svc.Name, <name>svc.PAP, <name>svc.PAP, "services"},
```

And add to the import block at the top:

```go
<name>svc "github.com/tbckr/trident/internal/services/<name>"
```

### Step 4 — Wire the command

Create `internal/cli/<name>.go` (one file per command — every existing service has its own: `cymru.go`, `crtsh.go`, `dns.go`, …). Pattern:

- For services that take a domain/IP/etc.: call `runServiceCmd` (it enforces the PAP gate).
- For aggregate services: call `runAggregateCmd` (enforces `MinPAP`).
- Identify-style services skip both and inline their own PAP check — wrap `services.ErrPAPBlocked` the same way `runServiceCmd` does.
- Always use the deps factories (`d.newHTTPClient()`, `d.newResolver()`, `d.loadPatterns()`, `d.papLevel`) — bypassing them breaks the proxy/UA configuration.
- Register the command in `internal/cli/root.go` (add to `newRootCmd`'s services group, alongside the others).

### Step 5 — Update README.md (all 5 places)

Find anchors by heading regex — line numbers drift as the README grows. Run this once to locate everything:

```bash
grep -nE '^## (Quickstart|Services|PAP System|Commands Reference)|^### (Project Structure|`)' README.md
```

| Section | Anchor | What to add |
|---------|--------|-------------|
| Quickstart bash block | `## Quickstart` | One-line `trident <name> example.com` example with a `# Description` comment |
| Services table | `## Services` | Row: `\| <name> \| <description> \| <PAP> \| <source> \|` (table appears directly under the heading) |
| PAP table | `## PAP System` | Add `<name>` to the `Permitted Services` cell for its PAP level (or higher) |
| Commands Reference | `## Commands Reference` | New `### \`<name>\` — <short title>` subsection with description, flags, example. Insert alphabetically. |
| Project Structure tree | `### Project Structure` (under `## Architecture` or similar) | Add `<name>/` line under `internal/services/` in the tree |

After each insertion, eyeball the surrounding context — these sections are hand-formatted Markdown tables and trees, easy to misalign.

### Step 6 — Verify locally

```bash
just lint                                # all golangci-lint v2 rules
just test                                # full suite
go test -cover ./internal/services/<name>/...   # confirm ≥80%
just ci                                  # everything (build + test + coverage + lint + vuln + flake)
```

If `just coverage` reports < 80% for the new package, add tests before committing.

### Step 7 — Open the PR

Per project convention:

```bash
git add internal/services/<name>/ internal/cli/services.go internal/cli/<name>.go internal/cli/root.go README.md
git commit -m "feat(<name>): add <name> service"
gh pr create --fill
```

Use `feat:` prefix (new feature → minor version bump on svu). For follow-up bugfixes use `fix(<name>):`.

## Commands Reference

| Purpose | Command |
|---|---|
| List existing services | `ls internal/services/` |
| Verify PAP constant ordering | `grep -n 'PAP = pap\.' internal/services/<name>/service.go` |
| Find README anchors | `grep -nF "$NEEDLE" README.md` |
| Run new service tests only | `go test -v -cover ./internal/services/<name>/...` |
| Full CI | `just ci` |

## Common Mistakes (and how the templates prevent them)

| Mistake | What the template does | Where in CLAUDE.md |
|---------|------------------------|---------------------|
| Two `%w` in one `fmt.Errorf` (multi-error) | Single sentinel + `%v` for inner err | "Other → fmt.Errorf single sentinel" |
| `resp.StatusCode` panic on transport error | `if resp.Response == nil` guard | "HTTP / req → req.Response nil guard" |
| `table.Header(...)` assigned to var | Bare call, no assignment | "Tables / output → tablewriter v1.1.3 API" |
| `runServiceCmd` skipped, custom Run inline | Comment block at top of service.go calling out the pattern | "Service Implementations" |
| Forgot `allServices()` entry | Step 3 of this checklist | "Key Gotchas → CLI" |
| Sort in `Run`, not `WriteTable` | TODO comment placed in `result.go` only | "Tables → display-only sort" |
| `httpclient.New` called from command file | Step 4 explicitly lists factories | "CLI / deps factory methods" |

## What this skill does NOT do

- It does NOT modify `cmd/trident/main.go` (commands are wired in `internal/cli/root.go`).
- It does NOT touch the `aliases` group — that's for user-defined aliases.
- It does NOT run `just release` or push anything.
- It does NOT update man pages — `cmd/docgen` regenerates them automatically during build.
