# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Trident** is a Go-based OSINT CLI tool (port of Python's [Harpoon](https://github.com/Te-k/harpoon)). Phase 1 MVP is implemented — three keyless OSINT services: DNS, ASN, and crt.sh. The PRD is in `docs/PRD.md`.

## Tools

**Library documentation:** Always use the context7 MCP (`mcp__plugin_context7-plugin_context7__resolve-library-id` + `mcp__plugin_context7-plugin_context7__query-docs`) to look up library docs. Never guess API shapes — fetch authoritative documentation first.

## Module
`github.com/tbckr/trident`

## Commands

Once the Go module is initialized, these commands apply:

```bash
# Build
go build ./...

# Run all tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# Run a single test
go test ./internal/services/... -run TestCrtshService -v

# Lint (strict)
golangci-lint run

# Run the CLI
go run ./cmd/trident/main.go dns example.com
go run ./cmd/trident/main.go asn AS15169
go run ./cmd/trident/main.go crtsh example.com
```

## Architecture

### Directory Structure (Phase 2)

```
cmd/trident/        # main.go — delegates immediately to run()
internal/
  cli/              # Cobra root command, global flags, output formatting
  config/           # Viper config loading (~/.config/trident/config.yaml)
  httpclient/       # req.Client factory with proxy + UA rotation
  pap/              # PAP level constants and Allows() enforcement
  worker/           # Bounded goroutine pool (pool.go) + stdin reader (stdin.go)
  services/         # One package per service (dns/, asn/, crtsh/, threatminer/, pgp/)
  output/           # Text (tablewriter), JSON, plain formatters + defang helpers
  validate/         # Shared input validators — IsDomain() and future helpers
```

**Per-service file layout** — every service package must follow this 4-file structure:
```
internal/services/<name>/
├── doc.go           # // Package <name> ... comment only
├── service.go       # Service struct, constructor, Name, PAP, Run, helpers
├── result.go        # Result struct + IsEmpty, WritePlain, WriteText methods
└── multi_result.go  # MultiResult struct + WriteText (embeds MultiResultBase)
```

**Per-service test file layout** — 3 test files mirror the 4 source files:
```
├── service_test.go      # TestRun_*, TestService_*, shared helpers (newTestClient, fixtures)
├── result_test.go       # TestResult_* only
└── multi_result_test.go # TestMultiResult_* only
```

### Core Patterns

**Dependency Injection:** Constructor injection everywhere. No global state or singletons.
```go
func NewDNSService(resolver DNSResolverInterface, logger *slog.Logger) *DNSService
func NewCrtshService(client *req.Client, logger *slog.Logger) *CrtshService
```

**`*req.Client` is a hard dependency** — not abstracted behind an interface. Mock HTTP in tests via `httpmock.ActivateNonDefault(client.GetClient())`.

**`httpclient.New()` signature:** `New(proxy, userAgent string, logger *slog.Logger, debug bool) (*req.Client, error)` — pass `nil, false` in tests/non-debug paths. When `debug && logger != nil`, attaches `OnAfterResponse` hook that logs timing + error body via `client.EnableTraceAll()`. Proxy precedence: explicit `proxy` arg → `client.SetProxyURL(proxy)`; empty `proxy` → `client.SetProxy(http.ProxyFromEnvironment)` (honours `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` env vars at request time). `req.Client.SetProxy` accepts `func(*http.Request) (*url.URL, error)` — same signature as `http.ProxyFromEnvironment`.

**`req.TraceInfo` fields** (context7 doesn't index this struct — fetch from `https://raw.githubusercontent.com/imroc/req/master/trace.go`): `DNSLookupTime`, `ConnectTime`, `TCPConnectTime`, `TLSHandshakeTime`, `TotalTime`. Access via `resp.TraceInfo()` in hook; request method/URL via `resp.Request.RawRequest.Method` / `.URL.String()`. `resp.String()` is safe in `OnAfterResponse` — req buffers the body; downstream readers are unaffected.

**tablewriter v1.1.3 API:** `table.Header([]string{...})` + `table.Bulk([][]string{...})` + `table.Render()`. Old `SetHeader`/`Append([]string)` don't exist — use `Bulk` for multi-row, `Append(any)` for single row.

**tablewriter header uppercasing:** `table.Header([]string{...})` renders all headers in ALL CAPS — test assertions must use uppercase strings: `"DOMAIN"` not `"Domain"`, `"FIRST SEEN"` not `"First Seen"`.

**tablewriter error returns:** Both `table.Bulk(rows)` and `table.Render()` return `error` — always propagate them; `errcheck` will fail the lint if ignored.

**`output.NewWrappingTable`** — shared factory in `internal/output/terminal.go`; use for plain (ungrouped) tables. **`output.NewGroupedWrappingTable`** — use when rows are grouped by a type column (e.g. DNS): merges repeated first-column cells (`MergeHierarchical`) and draws separator lines between groups (`BetweenRows: tw.On`); requires `"github.com/olekukonko/tablewriter/renderer"` imported in `terminal.go`. Overhead values: 20 for 2-column tables, 6 for 1-column tables.

**gosec G115 (`uintptr→int`)** — any `int(f.Fd())` call (e.g. `term.GetSize`, `term.IsTerminal`) always triggers G115. Suppress with `//nolint:gosec // uintptr→int is safe for file descriptors; they fit in int on all supported platforms`.

**`internal/testutil`** — `MockResolver` (implements `DNSResolverInterface` with optional `*Fn` fields) + `NopLogger()`. Import in `_test` files for DNS/ASN service tests.

**crtsh URL:** Use `"%%.%s"` (double `%%`) in the constant so `fmt.Sprintf` emits a literal `%.` before the domain. `"%.%s"` silently causes an arg-count mismatch.

**crtsh subdomain filter (`isValidSubdomain`):** Check wildcards first (`strings.HasPrefix(sub, "*")`), then root-domain equality, then suffix, then `validate.IsDomain`. Wildcard check must precede suffix — `*.example.com` passes `strings.HasSuffix(sub, ".example.com")` and won't be caught otherwise.

**crtsh test fixture:** `testdata/crtsh_response.json` contains `example.com` (root domain) as a deliberate filtered-case entry — assert `NotContains(t, result.Subdomains, "example.com")`, never `Contains`.

**`DNSResolverInterface`** — only DNS/ASN use an interface (for `*net.Resolver` mocking). Defined in `internal/services/interfaces.go`.

**`run` function pattern:** `main()` delegates to `run(ctx context.Context)` which accepts all dependencies and returns an error — enables testability.

**Signal handling & graceful cancellation** — `main()` creates `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` and passes the context to `cli.Execute(ctx, ...)`. After `run()` returns, `errors.Is(err, context.Canceled)` → silent `os.Exit(0)` (user cancellation is intentional, not an error). `Execute` signature: `Execute(ctx context.Context, stdout, stderr io.Writer) error`; uses `cmd.ExecuteContext(ctx)` so all subcommands receive context via `cmd.Context()`.

**`buildDeps` signature:** `buildDeps(cmd *cobra.Command, stderr io.Writer) (*deps, error)` — pass `cmd` so `config.Load` gets inherited persistent flags. Called once from root's `PersistentPreRunE`, not from individual subcommand `RunE` functions.

**`deps` struct fields:** `cfg *config.Config`, `logger *slog.Logger`, `doDefang bool` — no derived type-casts. Only multi-input computed values (like `doDefang`) belong in `deps`; inline simple casts at usage (`output.Format(d.cfg.Output)`, `pap.MustParse(d.cfg.PAPLimit)`).

**`resolveInputs` terminal guard** — when no args are given and stdin is an interactive terminal (`term.IsTerminal(int(f.Fd()))`), returns `fmt.Errorf("no input: pass an argument or pipe stdin")` immediately. Piped stdin still flows through `worker.ReadInputs`. `golang.org/x/term` is already in `go.mod`.

**CLI I/O wiring:** `Execute` calls `cmd.SetOut(stdout)` + `cmd.SetErr(stderr)` on root. Subcommand `RunE` uses `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` — Cobra walks the parent chain. Subcommand constructors: `newXxxCmd(d *deps) *cobra.Command` — no `io.Writer` param.

**`PersistentPreRunE` invariant:** Root's hook populates `var d deps` (declared in `newRootCmd` scope) before any subcommand `RunE` runs. Cobra only executes the innermost `PersistentPreRunE` — never add one to a subcommand without also calling `buildDeps` there.

**`completion` subcommand exception** — the `completion` command overrides root's `PersistentPreRunE` with a no-op to prevent `buildDeps` side effects (config dir creation) during shell completion generation. This is the only permitted exception to the invariant.

**Cobra command grouping** — `cmd.AddGroup(&cobra.Group{ID: "osint", Title: "OSINT Services:"})` then set `GroupID: "osint"` on each `*cobra.Command` struct. Currently: `"osint"` for OSINT services, `"utility"` for completion/version. Requires cobra v1.7+; project uses v1.10.2.

**`cmd.MarkFlagsMutuallyExclusive`** — works for persistent flags on root command (calls `mergePersistentFlags()` internally). Used for `"defang"` / `"no-defang"`; the `buildDeps` check remains as a defensive fallback.

**Shell completion generation** — `GenBashCompletionV2(w, true)`, `GenZshCompletion(w)`, `GenFishCompletion(w, true)`, `GenPowerShellCompletionWithDesc(w)` all return `error` — always propagate. Completion subcommand lives in `internal/cli/completion.go`.

**Version build variables** — `var version`, `commit`, `date` in `internal/cli/version.go`; injected at build time via `-X github.com/tbckr/trident/internal/cli.version=v1.0.0`. `cmd.Version = version` enables `trident --version`; `cmd.SetVersionTemplate("trident version {{.Version}}\n")` controls its format.

**`config.RegisterFlagCompletions(cmd)`** — call after `config.RegisterFlags(cmd.PersistentFlags())` in `newRootCmd`. Completion functions in `internal/config/completion.go`; `RegisterFlagCompletionFunc` returns `error` — discard with `_ =`.

**`services.ErrInvalidInput`** — unified validation sentinel in `internal/services/service.go`. New services must use this (not define their own `ErrInvalidInput`). Wrap with context: `fmt.Errorf("%w: must be …: %q", services.ErrInvalidInput, input)`.

**Type assertions in tests** — always use two-value form: `result, ok := raw.(*T); require.True(t, ok, "expected *T")`. Bare `raw.(*T)` panics on failure.

**`services.Result` interface** — defined in `internal/services/service.go`; requires `IsEmpty() bool`. Every service's `*Result` and `*MultiResult` satisfy it. Used by `runServiceCmd` to skip table rendering and log at INFO level.

**`MultiResult` pattern** — each service's `multi_result.go` embeds `services.MultiResultBase[Result, *Result]` (provides `IsEmpty`, `MarshalJSON`, `WritePlain`) and adds only `WriteText`. ThreatMiner overrides `WritePlain` (prefixes each record with `r.Input`). After embedding, init via assignment: `m := &dns.MultiResult{}; m.Results = [...]` — composite literal with promoted fields is a compile error.

**`services.MultiResultBase[T, PT]`** — generic base in `internal/services/multi.go`; `PT multiItem[T]` constrains the element type (`*T` + `IsEmpty() bool` + `WritePlain(io.Writer) error`). Provides `IsEmpty`, `MarshalJSON`, `WritePlain`; embed it and add `WriteText` to complete a service's `MultiResult`.

**`runServiceCmd`** — shared `RunE` body in `internal/cli/root.go`; handles PAP check, input resolution, single-result and bulk paths (calls `svc.AggregateResults(valid)` for 2+ valid results). Each subcommand's `RunE` just instantiates the service and calls `runServiceCmd(cmd, d, svc, args)`.

**CLI empty-result pattern** — after `svc.Run()` succeeds, each CLI command checks `IsEmpty()` and returns early without calling `writeResult()`:
```go
if ok && someResult.IsEmpty() {
    logger.Info("no … found", "input", args[0])
    return nil
}
```
`logger` comes from `buildDeps`; exit code is 0 (zero results is valid, not an error).

**PAP level ordering** — ascending activity: `RED(0) < AMBER(1) < GREEN(2) < WHITE(3)`. RED = non-detectable (offline/local); AMBER = 3rd-party APIs; GREEN = direct target interaction; WHITE = unrestricted. `Allows(limit, service Level) bool { return service <= limit }` — a service is blocked when its level exceeds the user's limit. Default `--pap=white` permits everything.

**`pap.MustParse`** — like `Parse` but panics on invalid input; safe to call in subcommands after `buildDeps` has already validated `cfg.PAPLimit`. Subcommands use `pap.MustParse(d.cfg.PAPLimit)` wherever a `pap.Level` is needed at call time.

**`output.ResolveDefang`** — defanging decision function in `internal/output/defang.go`; extracted from CLI so it can be unit-tested (since `internal/cli/` has intentional 0% coverage). Accepts `(papLevel, format, explicitDefang, noDefang)` and encodes all PAP-trigger rules.

**`DefangURL` host extraction** — never use `strings.Index(s, "/")` to find the host/path boundary; it hits the first `/` inside `://`. Find `://` first, skip 3 bytes, then search for the next `/` in `s[hostStart:]`.

**`gofmt` struct field alignment** — never pad struct field types/tags with extra spaces for visual alignment (e.g., `Input      string     \`json:"input"\``); gofmt normalizes to single-tab separation and `golangci-lint` will fail.

**PGP testdata workaround** — a pre-tool-use hook blocks creation of `.txt` files in `testdata/`; inline the MRINDEX fixture as a `const mrindexFixture` string directly in `service_test.go` instead.

**Service interface** — every service implements:
```go
type Service interface {
    Name() string
    PAP() pap.Level
    Run(ctx context.Context, input string) (Result, error)
    AggregateResults(results []Result) Result
}
```

### Service Implementations (Phase 2)

| Command | Implementation | PAP |
|---------|---------------|-----|
| `dns` | Go `net` package — A, AAAA, MX, NS, TXT records; canonical `WriteText` order: NS → A → AAAA → MX → TXT → PTR | GREEN (direct target interaction) |
| `asn` | Team Cymru DNS: IPv4 → `<reversed>.origin.asn.cymru.com`; IPv6 → 32-nibble reversal + `.origin6.asn.cymru.com`; ASN → `AS<n>.asn.cymru.com` | AMBER (3rd-party API) |
| `crtsh` | HTTP GET `https://crt.sh/?q=%.<domain>&output=json` via `imroc/req` | AMBER (3rd-party API) |
| `threatminer` | `https://api.threatminer.org/v2/{domain,host,sample}.php` — auto-detects domain/IP/hash input; status_code "404" → empty result (not error) | AMBER (3rd-party API) |
| `pgp` | `https://keys.openpgp.org/pks/lookup?op=index&options=mr` — HKP MRINDEX format; HTTP 404 → empty result (not error) | AMBER (3rd-party API) |

### Configuration

- File: `~/.config/trident/config.yaml` (created with `0600` permissions)
- Managed via `spf13/viper`
- Env vars take precedence; prefix: `TRIDENT_*`
- Respect XDG on Linux, AppData on Windows
- **Config API:** `config.RegisterFlags(cmd.PersistentFlags())` in root.go; `config.Load(cmd.Flags())` in `buildDeps`. Viper owns the full precedence chain — no scattered `if flag == "" {}` guards.
- **Flag→viper key discrepancies** (hyphen→underscore, rename): `--user-agent`→`user_agent`, `--pap`→`pap_limit`, `--no-defang`→`no_defang`. These drive mapstructure tags and env vars (`TRIDENT_USER_AGENT`, `TRIDENT_PAP_LIMIT`, `TRIDENT_NO_DEFANG`).
- **`Config.ConfigFile`** has no mapstructure tag — set manually after `v.Unmarshal(&cfg)` (meta-field, not a viper key).

### Global Flags (Phase 2)

| Flag | Default |
|------|---------|
| `--config` | `~/.config/trident/config.yaml` |
| `--verbose` / `-v` | Info level logging |
| `--output` / `-o` | `text` (also: `json`, `plain`) |
| `--concurrency` / `-c` | `10` |
| `--proxy` | — (supports `http://`, `https://`, `socks5://`) |
| `--user-agent` | rotating browser UAs |
| `--pap` | `white` |
| `--defang` | `false` |
| `--no-defang` | `false` |

### Tech Stack

- **CLI:** `spf13/cobra`
- **Config:** `spf13/viper`
- **HTTP:** `imroc/req` v3 (no external SDKs — all APIs implemented natively)
- **Logging:** `log/slog` (stdlib only — no zap/logrus)
- **Tables:** `olekukonko/tablewriter`
- **Tests:** `stretchr/testify` + `jarcoal/httpmock`
- **Lint:** `golangci-lint` v2 (strict — CI fails on any lint error). Config requires `version: "2"` at top; formatters (`gofmt`, `goimports`) go in `formatters:` section, not `linters:`. GitHub Action: `golangci/golangci-lint-action@v8` with `version: latest` (pinning a specific version risks Go version mismatch with `go.mod`).
- **GoReleaser v2 `format_overrides`** — use `format: zip` (singular scalar), not `formats: [zip]` (list); `goreleaser check` catches this at validation time.
- **golangci-lint v2 config structure:** `linters-settings` → `linters.settings`; `formatters-settings` → `formatters.settings`; `issues.exclude-rules` → `linters.exclusions.rules`. `goimports.local-prefixes` is an array (not a string). `gosimple` is merged into `staticcheck` — do not list it separately.
- **gosec suppressions:** `gosec.excludes` under `linters.settings` is unreliable; prefer `linters.exclusions.rules` with `text: "G304"` or an inline `//nolint:gosec // reason` comment. `nolintlint` will error if the directive is present but gosec doesn't fire on that line — remove unused nolint directives rather than suppressing them.
- **revive `package-comments`:** Every package must have a `// Package foo ...` comment in `doc.go` (never inline in an implementation file). New packages without this will fail lint.

## Key Constraints

- **No external I/O in tests** — all DNS and HTTP must be mocked; no real network calls. DNS: `mockResolver` struct; HTTP: `httpmock.ActivateNonDefault(client.GetClient())`.
- **No ad-hoc CLI runs for verification** — `go run ./cmd/trident/main.go <service> ...` may hit live endpoints; use `go build ./...` + `go test ./...` to verify changes instead.
- **Shell stderr noise** — every `go` command emits `alias: --: Nicht gefunden.` from shell init; this is harmless. Judge build/test success by explicit echo (`&& echo "BUILD OK"`), not absence of stderr.
- **Diagnostic lag after edits** — IDE diagnostics (DuplicateDecl, UndeclaredName) may show stale errors for several seconds after an Edit tool call. Always confirm actual state with `go build ./...` rather than re-editing based on stale diagnostics.
- **No `os/exec`** for DNS — use `net.Resolver` directly
- **Enforced HTTPS only** — no `InsecureSkipVerify`
- **Output sanitization** — strip ANSI escape sequences from external data before printing
- **80% minimum test coverage** — enforced on `./internal/services/...` only (CLI/cmd packages intentionally have 0%). CI uses `go test ./internal/services/... -coverprofile=svc_coverage.out`.
- **Cross-platform** — must compile on Linux, macOS, Windows; use `filepath.Join`

## CI/CD

**Workflow files:**
- `ci.yml` — push/PR: test (with `go mod verify` + tidy check), lint, govulncheck (plain `run` step — no sandbox)
- `release.yml` — tag push: GoReleaser + SBOM + Cosign
- `vuln-schedule.yml` — daily 06:00 UTC: govulncheck in gVisor sandbox
- `latest-deps.yml` — daily 07:00 UTC: `go get -u -t ./... && go test ./...` in gVisor sandbox

**SHA pinning:** all `uses:` lines are pinned by 40-char commit SHA. Dependabot (`.github/dependabot.yml`, weekly, `github-actions` ecosystem only) opens PRs when action authors release new versions. To look up a SHA when updating:
```bash
sha=$(gh api repos/ORG/REPO/git/ref/tags/vX --jq '.object.sha')
type=$(gh api repos/ORG/REPO/git/ref/tags/vX --jq '.object.type')
# If type == "tag" (annotated), dereference:
sha=$(gh api repos/ORG/REPO/git/tags/$sha --jq '.object.sha')
```

**`geomys/sandboxed-step`** — runs steps in a gVisor sandbox. Requires `persist-credentials: false` on the preceding `actions/checkout` step. Workspace changes don't persist unless `persist-workspace-changes: true` is set.

**Go module dependency policy** — Dependabot is intentionally NOT configured for the `gomod` ecosystem. `govulncheck` (call-graph reachability) and `latest-deps.yml` (freshness) replace Dependabot's noisy, reachability-unaware Go module PRs.

## Phase Roadmap

The full PRD is in `docs/PRD.md`. Phases deferred from MVP:
- **Phase 2:** ✅ Complete — stdin/bulk input, concurrency, proxy, PAP system, defanging, ThreatMiner, PGP; shell completions, command grouping, rich help text, version command (ldflags vars prepped for GoReleaser)
- **Phase 3:** GoReleaser, SBOM (CycloneDX), Cosign signing, rate limiting with jitter, `burn` command
