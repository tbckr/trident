# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Go-based OSINT CLI tool (port of Python's [Harpoon](https://github.com/Te-k/harpoon)). Services: DNS, Cymru, crt.sh, ThreatMiner, PGP, Quad9, detect, identify.

**Naming:** Always lowercase `trident` — never `Trident`.

**Git commits:** Never add `Co-Authored-By` or any attribution trailer to commit messages.

**Library docs:** Use context7 MCP (`resolve-library-id` + `query-docs`). Never guess API shapes.

## Module
`github.com/tbckr/trident`

## Commands
```bash
go build ./...
go test ./... -coverprofile=coverage.out
go test ./internal/services/... -run TestXxx -v
golangci-lint run
go mod tidy
```

## justfile Targets
- `just release [kind]` — `svu <kind>` (default: `next`; e.g. `major`/`minor`/`patch`/`prerelease`) → `git tag` → `git push` + `git push --tags`
- `just flake-update` — `nix flake update` (refreshes `flake.lock`)
- `just build` / `just test` / `just lint` — standard build, test, lint
- `just test-race` — run all tests with race detector
- `just test-pkg <pkg>` — verbose tests for a specific package
- `just fuzz <pkg>` — run fuzz tests for a specific package
- `just coverage` — check service coverage meets 80% threshold
- `just fmt` — format all Go files with gofmt
- `just ci` — run all CI checks locally (build → test → coverage → lint → semgrep → leaks → vuln → flake-check)
- `just tidy` / `just tidy-check` — tidy modules / verify they're clean
- `just vuln` / `just license-check` / `just flake-check` — govulncheck, license audit, nix check
- `just semgrep` — semgrep scan against `.semgrep/` + `p/golang` + `p/gosec` (requires `nix develop` or semgrep on PATH)
- `just leaks` — betterleaks secret scan against the working tree (requires `nix develop` or betterleaks on PATH)
- `just flake-build` — build the Nix package locally
- `just goreleaser-check` — validate `.goreleaser.yaml` config
- `just verify-release <archive>` — verify release artifact attestation
- `just upgrade` — upgrade direct dependencies (`scripts/deps.sh upgrade`, every direct module `@latest` — it ignores holds: set modules with a `// Revisit` note on their go.mod require line back with `go get <mod>@<ver>`, and check the go directive did not move); then update `vendorHash`, `git add go.mod go.sum flake.nix` (tidy-check diffs against the index) and run `just ci`
- `just harden-repo` — apply repository hardening settings via `gh` API
- `just outdated` — list what `just upgrade` leaves out, for review (never fails): newer releases/deprecations of direct modules (not new majors — `go list -m -u` stays within a module path), Go release support, hand-pinned tools/images, flake input age, `Revisit`/`Drop this once` notes

## Architecture

```
cmd/trident/        # main.go → run()
cmd/docgen/         # man pages + shell completions generator (cobra/doc); used by GoReleaser & Nix flake
internal/
  cli/              # Cobra root, global flags, output; deps.go has deps struct + factory methods
  config/           # Viper config (~/.config/trident/config.yaml)
  httpclient/       # req.Client factory (proxy + UA)
  input/            # stdin line reader
  pap/              # PAP level constants + Allows()
  doh/              # DNS-over-HTTPS (RFC 8484, Quad9)
  ratelimit/        # Token-bucket rate limiter with ±20% jitter
  resolver/         # *net.Resolver factory (SOCKS5 leak prevention)
  worker/           # Bounded goroutine pool
  services/         # One package per service; IsDomain() here
  appdir/           # OS config-dir helpers: ConfigDir(), EnsureFile()
  apperr/           # Shared error sentinels (leaf; no internal imports)
  detect/           # Provider detection: CDN/Email/DNS/TXT (pure, no I/O); patterns.yaml embedded
  output/           # Table/JSON/text formatters + defang helpers
  testutil/         # Shared test helpers (mock resolver, nop logger)
  version/          # Build version info (ldflags + BuildInfo fallback)
```

**Per-service file layout** (required for all new services):
```
internal/services/<name>/
├── doc.go           # // Package <name> ... comment only
├── service.go       # Service struct, constructor, Name, PAP, Run, helpers
├── result.go        # Result struct + IsEmpty, WriteText, WriteTable
└── multi_result.go  # MultiResult + WriteTable (omit if no bulk path)
```
Test files: `service_test.go`, `result_test.go`, `multi_result_test.go`.

Every service exports package-level `Name` and `PAP` constants; aggregate services also export `MinPAP`. `const PAP = pap.AMBER` is valid Go — `pap.Level` is an iota-typed integer.

**Services that don't implement `services.Service`** (e.g. `identify` — typed slice inputs): use custom `Run` signature, call `writeResult` from `RunE`, inline PAP check. No `runServiceCmd`, no `AggregateResults`, no `multi_result.go`. **Inline PAP check (identify-style)** — must still wrap `services.ErrPAPBlocked`: `fmt.Errorf("%w: %q requires PAP %s but limit is %s", services.ErrPAPBlocked, svc.Name(), svc.PAP(), d.papLevel)` — same format as `runServiceCmd`.

## Service Implementations

| Command | PAP | Notes |
|---------|-----|-------|
| `dns` | GREEN | A, AAAA, MX, NS, TXT; WriteTable order: NS→A→AAAA→MX→TXT→PTR |
| `cymru` | AMBER | IPv4: `<reversed>.origin.asn.cymru.com`; IPv6: 32-nibble+`.origin6`; ASN: `AS<n>.asn.cymru.com` |
| `crtsh` | AMBER | URL constant: use `"%%.%s"` (double `%%`) so `fmt.Sprintf` emits literal `%.domain` |
| `threatminer` | AMBER | Auto-detects domain/IP/hash; status_code "404" → empty result (not error) |
| `pgp` | AMBER | HKP MRINDEX; HTTP 404 → empty result; any HKP query accepted (email/name/`0x`-prefix) |
| `quad9 resolve` | AMBER | RFC 8484 DoH; A, AAAA, NS, MX, TXT; partial result on context cancel |
| `quad9 blocked` | AMBER | blocked = NXDOMAIN + empty authority section; genuine NXDOMAIN has SOA |
| `apex` | AMBER | Aggregate service (`runAggregateCmd`); composes Quad9 DoH + Cymru ASN. Exports `MinPAP` alongside `PAP` |
| `detect` | GREEN | CNAME/MX/NS/TXT; import detect pkg as `providers "...internal/detect"` to avoid name collision |
| `identify` | RED | Pure pattern matching; custom Run signature; no `multi_result.go` |

**README.md updates** — when adding a service, update 5 places: quick-start block, Services table, PAP table, Commands Reference section, architecture tree.

**`allServices()` in `internal/cli/services.go`** — must be updated when adding a service; reads package-level constants (no instantiation). Easy to miss since it's not near the service code.

## Key Gotchas

### CLI / deps factory methods
**New commands must use `deps` factory methods** — use `d.newHTTPClient()`, `d.newResolver()`, `d.loadPatterns()`, and read `d.papLevel` directly. Factories + `papLevel` live in `internal/cli/deps.go`. (Semgrep enforces the `httpclient.New` / `resolver.NewResolver` / `pap.MustParse(d.cfg.PAPLimit)` cases — `loadPatterns` is not yet codified.)

### HTTP / req
**`req.Client` retry API** — client-level: `SetCommonRetryCount`, `AddCommonRetryCondition`, `SetCommonRetryInterval`. Bare `SetRetryCount`/`AddRetryCondition` are request-level only (`*req.Request`), not on `*req.Client`. Since req v3.60.0 an error returned by an `OnBeforeRequest` hook also goes through the retry conditions — trident's transport-error condition (`internal/httpclient/ratelimit.go`) would retry it; keep hooks returning only `ctx.Err()` or narrow the condition before adding a validating hook.

**Proxy bypass in req (imroc/req#206, #541)** — req ignores the proxy on its HTTP/2 paths: forced HTTP/2 never uses it, and after a proxied HTTP/2 connection dies req redials the origin directly (verified for CONNECT and SOCKS5). `httpclient.New` therefore forces HTTP/1.1 whenever a proxy is configured (flag/config or `HTTP(S)_PROXY`/`ALL_PROXY`; idle pool raised to 16 per host so bursts reuse tunnels) and wraps `DialContext` in `guardProxyBypass`, which refuses (`apperr.ErrProxyBypass`, never retried) any dial to a host the proxy should reach — every req TCP dial, proxied or not, passes through `DialContext`. An env `socks4://` proxy is refused too (req resolves target names locally for SOCKS4). For HTTP/2 use `httpclient.ForceHTTP2(client, proxy)` (quad9/apex: one multiplexed connection instead of a dial per concurrent request; no-op behind a proxy). Never call `EnableForceHTTP2`/`EnableHTTP3`/`SetDial`/`SetResolver`/`SetHosts`/`SetProxy*` on a client outside `internal/httpclient` (semgrep `trident-req-proxy-safe-transport`). The tests in `internal/httpclient/proxy_test.go` (loopback CONNECT + SOCKS5 proxy, tunnel kill) must keep passing; they fail on plain req. Drop the HTTP/1.1 forcing only once a req release contains #541 (Revisit note in `client.go`).

**httpmock patterns** — regex match: `"=~^"+baseURL`; transport failure (nil `resp.Response`): `httpmock.NewErrorResponder(err)` (not a 500 response).

### DNS / resolver
**`proxy.FromEnvironment()` caching** — uses `sync.Once`; `t.Setenv` has no effect after first call in a test binary. Read proxy vars with `os.Getenv` directly in `NewResolver`.

**`resolver.NewResolver` naming** — use `r`, not `resolver` (shadows the package import → compile error).

**`codeberg.org/miekg/dns`** — moved from `github.com/miekg/dns` to Codeberg; NOT a v2 module (no `/v2` suffix in import path). RR struct literals in tests need `codeberg.org/miekg/dns/rdata` import. `rdata.TXT{Txt: []string{...}}` (slice, not string). Set `m.Response = true` in test response messages. In `dns.TXT{}` struct literals, the embedded rdata field is `TXT:` (uppercase) not `Txt:`.

### Tables / output
**tablewriter v1.1.5 API** — `table.Header([]string{...})` is **void** (no return); `table.Bulk(rows)` and `table.Render()` return `error` — always propagate. `Header()` renders ALL CAPS — test assertions must use `"DOMAIN"`, not `"Domain"`. Since v1.1.4 a `\t` in a cell is expanded to `twwidth.TabWidth()` spaces, which depends on env (`TABWIDTH`, `TS`, `VIM_TABSTOP`), `.editorconfig` and the cwd — never golden-test cells containing tabs. Bump `olekukonko/ll` only together with tablewriter (ll ≥ v0.1.5 logs to stdout by default; only tablewriter ≥ v1.1.5 disables it).

**`output.NewGroupedWrappingTable`** — merging is **consecutive-only**; sort rows before `Bulk` when parallel queries may interleave records for the same host. **`NewGroupedWrappingTablePerCol`** — use when host column has variable-length content; `Global` cap applies to all columns equally, causing overflow if host is wide.

**Display-only sort** — sort a copy inside `WriteTable` (not in `Run`/`AggregateResults`) to keep JSON/text output order stable. Sort key: `"0:"` for primary input, `"1:"` for regular records, `"2:"` for sentinel rows.

### CLI / Cobra
**`PersistentPreRunE` invariant** — Cobra runs only the innermost `PersistentPreRunE`. Never add one to a subcommand without calling `buildDeps` there; it silently breaks the dependency injection chain. Exception: `completion` command overrides with a no-op.

**Cobra command groups** — groups render in registration order. The conditional `"aliases"` group must be registered positionally inside `newRootCmd` (not from `Execute()`) to maintain order: `"services"` → `"aliases"` → `"utility"`. `cmd.SetHelpCommandGroupID("utility")` assigns Cobra's built-in `help` to a group.

### Configuration
**Flag→viper key discrepancies** — hyphens become underscores: `--pap-limit` → `pap_limit`, `--user-agent` → `user_agent`, `--no-defang` → `no_defang`. These drive env vars (`TRIDENT_PAP_LIMIT`, etc.).

**`Config.ConfigFile`** has no mapstructure tag — set manually after `v.Unmarshal(&cfg)`.

**YAML nested map type assertion** — `yaml.Unmarshal` into `map[string]any` produces `map[string]any` for nested maps, not `map[string]string`. Always assert inner maps as `map[string]any`.

### Apex / detect service
**Apex detection pipeline** — pass all records of each type to detectors; never pre-filter by `Host==domain` (silently drops subdomain records like `_dmarc`, `_mta-sts`). All detection rows use `Host: "detected"`.

**`sortDetections()`** — Email detections arrive from both MX (`EmailProvider`) and TXT (`TXTRecord`), so they're non-consecutive; `sortDetections()` in `result.go` sorts by `(Type, Source, Provider)` before table render. Do not sort in `Run()`.

**`detect.Detection.Source`** — `"cname"` (CDN), `"mx"` (EmailProvider), `"ns"` (DNSHost), `"txt"` (TXTRecord). `TXTRecord` produces both `TypeEmail` and `TypeVerification`, both with `Source: "txt"`.

**`internal/detect` API** — detection methods are on `*Detector`, not package-level. Always `detect.NewDetector(patterns)` first. Patterns: `LoadPatterns(DefaultPatternPaths()...)` in CLI; `LoadPatterns()` (no args = embedded) in tests. User override: `~/.config/trident/detect.yaml`; reserved download path: `detect-downloaded.yaml`.

**detect/identify/apex `NewService` signature** — all three accept `patterns detect.Patterns` as final arg. Test helper: `func embeddedPatterns(t *testing.T) providers.Patterns { p, _ := providers.LoadPatterns(); return p }`. Use a `t.Helper()` wrapper; avoid `var`-init (panics on embed failure are invisible).

**`internal/detect` TXT matching** — `TXTRecord()` uses `strings.Contains` (substring), not suffix match.

### Lint
**golangci-lint v2 config keys** — `linters-settings` → `linters.settings`; `formatters-settings` → `formatters.settings`; `issues.exclude-rules` → `linters.exclusions.rules`. `gosimple` is merged into `staticcheck` — do not list separately.

**gosec G304** — fires on `os.Open`/`os.OpenFile` with variable paths sourced from user input, NOT on `os.ReadFile` and NOT on internal/trusted path parameters. Add `//nolint:gosec` only if the linter actually fires — unused directives are caught by `nolintlint`.

**gosec G115** — `int(f.Fd())` always triggers; suppress with `//nolint:gosec // uintptr→int is safe for fd`.

**revive stutter rule** — exported identifiers must not start with the package name (`detect.CDN` not `detect.DetectCDN`). When renaming would conflict with a constant, prefix constants with `Type` (`TypeCDN`, `TypeEmail`).

**revive `package-comments`** — every package needs `// Package foo ...` in `doc.go`, never inline in an implementation file.

**`strings.CutPrefix`** — `stringscutprefix` lint rule fires on `HasPrefix`+`TrimPrefix` combos. Always use `if v, ok := strings.CutPrefix(s, p); ok { ... }`.

**staticcheck QF1002** — use tagged `switch x { case y: }`, not typeless `switch { case x == y: }`.

**`gofmt` alignment** — never manually align struct field types or map key→value pairs (lint fails). `const` blocks may be pre-aligned (gofmt preserves it). No trailing blank line before closing `)`.

**Semgrep custom rules** — `.semgrep/` is the single source of truth for project-specific gotchas that can be expressed as a pattern. When you encounter a new codifiable gotcha, add a rule in `.semgrep/` (rule message addressed to the AI assistant, with full rationale) rather than just documenting it here; this file should only describe things Semgrep cannot enforce. Existing rule IDs: `trident-no-double-percent-w`, `trident-req-response-nil-guard`, `trident-no-direct-httpclient-new`, `trident-no-direct-resolver-new`, `trident-no-papmustparse-in-services`, `trident-httpmock-activate-scoped`, `trident-strings-index-url-slash`, `trident-waitgroup-use-go`, `trident-req-proxy-safe-transport`. Bypass with inline `// nosemgrep: <rule-id> -- reason`. Registry packs `p/golang` + `p/gosec` also run; suppress justified findings the same way.

### Other
**Context cancel in HTTP services** — check `errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)` *before* wrapping as `ErrRequestFailed`; return a partial/empty result instead. Pattern: crtsh, quad9, threatminer, pgp.

**`MultiResult` composite literal** — embeds `services.MultiResultBase[Result, *Result]`; init via field assignment (`m := &dns.MultiResult{}; m.Results = [...]`), not a composite literal (promoted fields cause compile error). `MarshalJSON` → bare JSON array, not envelope.

**Error sentinels** — defined in `internal/apperr/` (leaf; no internal imports); re-exported from `services` for backward compat. Use `services.ErrInvalidInput/ErrRequestFailed/ErrPAPBlocked` in service code. `doh` imports `apperr` directly to avoid the `doh → services` cycle. Wrap: `fmt.Errorf("%w: ...: %q", services.ErrInvalidInput, input)`. Never define per-service variants.

**`config.DefaultPatternsURL`** — built-in patterns download URL lives in `internal/config` (not `detect`). **`config.NormalizeKey`** — exported; converts `"pap-limit"` → `"pap_limit"`; don't duplicate locally.

**PGP testdata workaround** — a pre-tool-use hook blocks `.txt` in `testdata/`; inline MRINDEX fixtures as `const` strings in `service_test.go`.

**PAP level ordering** — `RED(0) < AMBER(1) < GREEN(2) < WHITE(3)`. Service is blocked when its level exceeds the user's limit. Default `--pap-limit=white` permits everything.

**`runAggregateCmd`** — use for `AggregateService` implementations (enforces `MinPAP` gate). `runServiceCmd` enforces `PAP` gate. Both error: `%q requires PAP %s but limit is %s`.

**detect API refactor scope** — when renaming package-level detect functions to methods, search ALL packages for callers (`grep -r "detect\."` or `grep -r "providers\."`) — not just the planned files. `apex` and `identify` both used `detect.CDN()` etc. and were missed in the plan's files summary.

## Configuration

File: `~/.config/trident/config.yaml` (0600). Env prefix: `TRIDENT_*`. Flag→viper key: hyphens become underscores. `Config.Aliases` uses mapstructure tag `"alias"`; file-only (no flag/env). `config.LoadAliases(path)` reads only alias section; returns empty non-nil map when file missing.

**`config.Load()` — custom path must exist** — `EnsureFile` and not-found silencing only apply to the default path; `--config=<file>` that doesn't exist returns an error. Tests using `newTestFlags(t, cfgFile)` must pre-create the file with `os.WriteFile(cfgFile, []byte{}, 0o600)`.

**Adding a persistent config flag** — 6 coordinated changes: (1) entry in `configKeys` map, (2) field in `Config` struct with `mapstructure` tag, (3) flag in `RegisterFlags`, (4) `BindPFlag` in `Load`, (5) completion func in `root.go` for enum keys, (6) `case` in `effectiveValue()` in `internal/cli/config.go` (omitting causes `config show`/`config get` to silently return empty). Tests: add to `TestParseValue` table + a `TestLoad_*` func.

**Resolve functions in domain packages** — `httpclient.ResolveProxy(proxy)` (env-var scanning), `detect.ResolvePatternFile(file)` (file-existence scanning). `effectiveValue()` in `cli/config.go` calls these directly — no CLI wrapper functions.

## Tech Stack

- **CLI:** cobra v1.10.2 | **Config:** viper | **HTTP:** imroc/req v3 | **Tables:** olekukonko/tablewriter v1.1.5
- **DNS wire:** `codeberg.org/miekg/dns` (NOT github.com/miekg/dns; no `/v2` suffix) | **Tests:** testify + httpmock
- **Lint:** golangci-lint v2 (strict; CI fails on any error)

**GoReleaser v2:** `format_overrides` uses `formats: [zip]` (list, not scalar). nfpms: use `ids:` not `builds:`. Before hooks: write output to `.build/` not `dist/`. No glob for transitive license trees in nfpms.

## Key Constraints

- **No external I/O in tests** — mock DNS (`internal/testutil.MockResolver`) and HTTP (`httpmock.ActivateNonDefault(client.GetClient())`)
- **No ad-hoc CLI runs** — `go run ./cmd/trident/main.go` may hit live endpoints; use `go build ./...` + `go test ./...`
- **Shell stderr noise** — `alias: --: Nicht gefunden.` is harmless; verify success with explicit echo (`&& echo "BUILD OK"`)
- **Diagnostic lag** — IDE diagnostics lag after edits; always confirm with `go build ./...`
- **80% coverage** — enforced on `./internal/services/...` only; CLI/cmd packages intentionally 0%
- **HTTPS only** — no `InsecureSkipVerify`
- **Output sanitization** — strip ANSI escape sequences from external data before printing
- **`internal/version` BuildInfo fallback** — `init()` reads `debug.ReadBuildInfo()` to populate Version/Commit/Date when ldflags aren't set (e.g. `go install`); ldflags always win. Logic lives in `applyBuildInfo(*debug.BuildInfo)` (exported for unit tests). Strips `v` prefix; skips `""` and `"(devel)"` for Version; truncates `vcs.revision` to 7 chars.
- **`.githooks/pre-commit`** — `nix develop` shellHook automatically sets `core.hooksPath=.githooks` so betterleaks scans staged changes before every commit. Hook is skip-when-missing (logs a warning and exits 0 if `betterleaks` not on PATH); CI workflow `betterleaks.yml` is the authoritative backstop. **betterleaks ≥ 1.10 does not auto-load `.betterleaks.toml`** — every invocation (workflow, `just leaks`, hook) passes `--config=.betterleaks.toml`; without it the allowlists are ignored and the scan fails on the fuzz seed corpus.

## CI/CD

- `ci.yml` — test + lint + govulncheck + license-check + nix flake check (push/PR)
- `release.yml` — GoReleaser + SBOM + VEX + GitHub Artifact Attestation (tag push); permissions scoped to job level (`contents: write`, `id-token: write`, `attestations: write`)
  - `actions/attest-build-provenance` attests every artifact in `checksums.txt` via GitHub Artifact Attestation (recognized by OpenSSF Scorecard)
  - Verification script: `scripts/verify-release.sh <ARCHIVE>` — runs `gh attestation verify`
- `goreleaser-lint.yml` — `goreleaser check` on `.goreleaser.yaml` changes (push/PR)
- `vuln-schedule.yml` — daily (06:00 UTC): govulncheck in sandboxed step
- `codeql.yml` — CodeQL SAST analysis for Go (push/PR to main + weekly Mon 06:00 UTC)
- `semgrep.yml` — Pattern-based SAST: custom rules in `.semgrep/` + `p/golang` + `p/gosec`; runs in `semgrep/semgrep` container (SHA-pinned); push/PR + weekly Mon 06:00 UTC; SARIF → Security tab
- `betterleaks.yml` — Secret scanning: full git-history scan in SHA-pinned `ghcr.io/betterleaks/betterleaks` container; push/PR + weekly Mon 06:00 UTC; SARIF → Security tab. Backstop for the `.githooks/pre-commit` developer hook.
- `scorecard.yml` — weekly (Mon 06:00 UTC): OpenSSF Scorecard → SARIF upload to Security tab
- `latest-deps.yml` — weekly (Mon 06:00 UTC), commits nothing: `go` job runs `scripts/deps.sh upgrade` (direct deps only: `go get <pkg>@latest` + `go mod tidy`) + `go test` in a sandbox on the **stable** Go toolchain (a dep needing newer Go raises the go directive in the logged diff instead of failing weekly); `go-support` job fails once go.mod's Go release drops out of support; `outdated` job writes `scripts/deps.sh outdated` to the run's step summary (never fails — no issue, no label). **Never `go get -u`** — upgrades all transitive deps, breaking direct-only intent.

Scripts: `scripts/harden-repo.sh` (idempotent repo hardening via `gh` API), `scripts/deps.sh` (`outdated` / `upgrade` / `go-support`), `scripts/verify-release.sh` (release verification).

All `uses:` lines are SHA-pinned. Dependabot covers github-actions only — monthly, one grouped PR, 7-day cooldown (not gomod — `govulncheck` handles reachability, `latest-deps.yml` handles freshness).

**Hand-pinned tools** — bumped by hand from `just outdated`, which parses these exact forms (keep them): `GOLANGCI_LINT_VERSION: vX.Y.Z` (ci.yml env; keep at the dev shell's nixpkgs version), `GORELEASER_VERSION: vX.Y.Z` (release.yml + goreleaser-lint.yml env), `govulncheck@vX.Y.Z` / `go-licenses/v2@vX.Y.Z` (ci.yml, vuln-schedule.yml, .goreleaser.yaml), `image: <repo>:<tag>@sha256:<digest>` (semgrep.yml, betterleaks.yml — tag and digest move together; `outdated` prints the new pin, and flags a same-tag digest change as "rebuilt"). All pins of one tool must agree; `outdated` reports disagreement. Never touch the `uses:` SHA or its `# vX.Y.Z` comment when bumping a tool.

## Nix Flake

**`vendorHash` must be updated** when `go.mod`/`go.sum` change — set to `pkgs.lib.fakeHash`, run `nix build`, extract hash from error output.

**`buildGo126Module`** — pinned to Go 1.26; bump to `buildGo1XXModule` when `go.mod` version changes. Dev shell uses `go_1_26` correspondingly.

**`.go-version`** — the Go release for the `actions/setup-go` steps (`go-version-file: .go-version`; the build/lint/scan/release steps add `check-latest: true` → newest patch). Exception: latest-deps.yml's `go` job uses `go-version: stable` on purpose — don't switch it to `.go-version`. Holds `X.Y` only, never a patch and never `go.mod` as the source: `go get` writes a patch-level go directive (`go 1.26.0`) whenever a dependency declares one (golang.org/x/* do), and setup-go would then install exactly X.Y.0 (stdlib CVEs, govulncheck fails). Raise it together with go.mod's go directive and the flake's `buildGo1XXModule`/`go_1_XX`; `just outdated` reports a mismatch. Do not add a `toolchain` line to go.mod instead.

**Checking nixpkgs Go version** — `nix eval "github:NixOS/nixpkgs/nixpkgs-unstable#go_1_26.version"` checks the channel the flake follows (independent of local `flake.lock`). Nixpkgs typically lags Go releases by a few days to ~2 weeks.

**`nix build` requires staged files** — new/modified files must be `git add`-ed before `nix build` (flakes only see git-tracked content).

**Nix flake ldflags** — only `Commit` and `Date` are set via ldflags; `Version` is intentionally omitted (defaults to `"dev"`) because flakes can't access git tags. `self.lastModifiedDate` provides the build timestamp (sliced to ISO 8601). The `version` binding (shortRev) is still used for the Nix store path and man page Source field.

**`postBuild` runs `cmd/docgen`** — generates man pages into `$TMPDIR/docs/man/`; `postInstall` installs them via `installManPage`. Changes to `cmd/docgen` or cobra command structure affect man page output in both GoReleaser and Nix.
