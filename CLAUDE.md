# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Trident** is a Go-based OSINT CLI tool (port of Python's [Harpoon](https://github.com/Te-k/harpoon)). Five keyless OSINT services are implemented: DNS, ASN, crt.sh, ThreatMiner, and PGP.

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

# Sync go.mod/go.sum after adding/removing imports
go mod tidy

# Run the CLI
go run ./cmd/trident/main.go dns example.com
go run ./cmd/trident/main.go asn AS15169
go run ./cmd/trident/main.go crtsh example.com
```

## Architecture

### Directory Structure

```
cmd/trident/        # main.go — delegates immediately to run()
internal/
  cli/              # Cobra root command, global flags, output formatting
  config/           # Viper config loading (~/.config/trident/config.yaml)
  httpclient/       # req.Client factory with proxy + UA rotation
  input/            # Line reader from io.Reader — Read() used by CLI stdin path
  pap/              # PAP level constants and Allows() enforcement
  resolver/         # *net.Resolver factory with SOCKS5 DNS-leak prevention
  worker/           # Bounded goroutine pool (pool.go only)
  services/         # One package per service (dns/, asn/, crtsh/, threatminer/, pgp/); IsDomain() lives here
  output/           # Text (tablewriter), JSON, plain formatters + defang helpers
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

**`httpclient.defaultUserAgent`** — package-level `var` (not `const`) in `internal/httpclient/client.go` that concatenates `version.Version` at runtime; imports `internal/version`. Single source of truth for the default UA — do not add a UA override in `buildDeps`.

**`req.TraceInfo` fields** (context7 doesn't index this struct — fetch from `https://raw.githubusercontent.com/imroc/req/master/trace.go`): `DNSLookupTime`, `ConnectTime`, `TCPConnectTime`, `TLSHandshakeTime`, `TotalTime`. Access via `resp.TraceInfo()` in hook; request method/URL via `resp.Request.RawRequest.Method` / `.URL.String()`. `resp.String()` is safe in `OnAfterResponse` — req buffers the body; downstream readers are unaffected.

**`req.Client` retry API** — client-level retry uses `SetCommonRetryCount(n)`, `AddCommonRetryCondition(func(*req.Response, error) bool)`, `SetCommonRetryInterval(func(*req.Response, attempt int) time.Duration)`. The bare `SetRetryCount`/`AddRetryCondition`/`SetRetryHook` are request-level only (`*req.Request`), not on `*req.Client`.

**`golang.org/x/net/proxy` SOCKS5 dialer** — `proxy.SOCKS5("tcp", host, nil, proxy.Direct)` returns a value that satisfies `proxy.ContextDialer`; type-assert with `dialer.(proxy.ContextDialer)` to get `.DialContext` for use in `net.Resolver.Dial`.

**`resolver.NewResolver` caller convention** — use `r` (not `resolver`) as the local variable name; naming it `resolver` shadows the package import and causes a compile error.

**tablewriter v1.1.3 API:** `table.Header([]string{...})` + `table.Bulk([][]string{...})` + `table.Render()`. Old `SetHeader`/`Append([]string)` don't exist — use `Bulk` for multi-row, `Append(any)` for single row.

**tablewriter header uppercasing:** `table.Header([]string{...})` renders all headers in ALL CAPS — test assertions must use uppercase strings: `"DOMAIN"` not `"Domain"`, `"FIRST SEEN"` not `"First Seen"`.

**tablewriter error returns:** `table.Header([]string{...})` is **void** (no return value). Only `table.Bulk(rows)` and `table.Render()` return `error` — always propagate those; `errcheck` will fail the lint if ignored.

**`output.NewWrappingTable`** — shared factory in `internal/output/terminal.go`; use for plain (ungrouped) tables. **`output.NewGroupedWrappingTable`** — use when rows are grouped by a type column (e.g. DNS): merges repeated first-column cells (`MergeHierarchical`) and draws separator lines between groups (`BetweenRows: tw.On`); requires `"github.com/olekukonko/tablewriter/renderer"` imported in `terminal.go`. Overhead values: 20 for 2-column tables, 6 for 1-column tables.

**gosec G115 (`uintptr→int`)** — any `int(f.Fd())` call (e.g. `term.GetSize`, `term.IsTerminal`) always triggers G115. Suppress with `//nolint:gosec // uintptr→int is safe for file descriptors; they fit in int on all supported platforms`.

**`internal/testutil`** — `MockResolver` (implements `DNSResolverInterface` with optional `*Fn` fields) + `NopLogger()`. Import in `_test` files for DNS/ASN service tests.

**crtsh URL:** Use `"%%.%s"` (double `%%`) in the constant so `fmt.Sprintf` emits a literal `%.` before the domain. `"%.%s"` silently causes an arg-count mismatch.

**crtsh subdomain filter (`isValidSubdomain`):** Check wildcards first (`strings.HasPrefix(sub, "*")`), then root-domain equality, then suffix, then `services.IsDomain`. Wildcard check must precede suffix — `*.example.com` passes `strings.HasSuffix(sub, ".example.com")` and won't be caught otherwise.

**crtsh test fixture:** `testdata/crtsh_response.json` contains `example.com` (root domain) as a deliberate filtered-case entry — assert `NotContains(t, result.Subdomains, "example.com")`, never `Contains`.

**`DNSResolverInterface`** — only DNS/ASN use an interface (for `*net.Resolver` mocking). Defined in `internal/services/interfaces.go`.

**`run` function pattern:** `main()` delegates to `run(ctx context.Context)` which accepts all dependencies and returns an error — enables testability.

**Signal handling & graceful cancellation** — `main()` creates `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` and passes the context to `cli.Execute(ctx, ...)`. After `run()` returns, `errors.Is(err, context.Canceled)` → silent `os.Exit(0)` (user cancellation is intentional, not an error). `Execute` signature: `Execute(ctx context.Context, stdout, stderr io.Writer) error`; uses `cmd.ExecuteContext(ctx)` so all subcommands receive context via `cmd.Context()`.

**`buildDeps` signature:** `buildDeps(cmd *cobra.Command, stderr io.Writer) (*deps, error)` — pass `cmd` so `config.Load` gets inherited persistent flags. Called once from root's `PersistentPreRunE`, not from individual subcommand `RunE` functions.

**Alias pre-expansion in `Execute()`** — aliases are loaded via `config.LoadAliases(peekConfigFlag(os.Args[1:], defaultPath))` *before* constructing the command, then passed to `newRootCmd(aliases)`. If `os.Args[1]` matches an alias name, `cmd.SetArgs(expanded)` rewrites args. The "aliases" group and stub commands are registered inside `newRootCmd` only when `len(aliases) > 0` — omitting the group hides it from `--help`. `peekConfigFlag` scans for `--config <path>` / `--config=<path>`; uses `strings.CutPrefix`.

**YAML nested map type assertion** — `yaml.Unmarshal` into `map[string]any` produces `map[string]any` for nested maps (not `map[string]string`). Always type-assert inner maps as `map[string]any`: `aliasMap, _ := raw["alias"].(map[string]any)`.

**`deps` struct fields:** `cfg *config.Config`, `logger *slog.Logger`, `doDefang bool` — no derived type-casts. Only multi-input computed values (like `doDefang`) belong in `deps`; inline simple casts at usage (`output.Format(d.cfg.Output)`, `pap.MustParse(d.cfg.PAPLimit)`).

**`resolveInputs` terminal guard** — when no args are given and stdin is an interactive terminal (`term.IsTerminal(int(f.Fd()))`), returns `fmt.Errorf("no input: pass an argument or pipe stdin")` immediately. Piped stdin still flows through `input.Read` (from `internal/input`). `golang.org/x/term` is already in `go.mod`.

**CLI I/O wiring:** `Execute` calls `cmd.SetOut(stdout)` + `cmd.SetErr(stderr)` on root. Subcommand `RunE` uses `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` — Cobra walks the parent chain. Subcommand constructors: `newXxxCmd(d *deps) *cobra.Command` — no `io.Writer` param.

**`PersistentPreRunE` invariant:** Root's hook populates `var d deps` (declared in `newRootCmd` scope) before any subcommand `RunE` runs. Cobra only executes the innermost `PersistentPreRunE` — never add one to a subcommand without also calling `buildDeps` there.

**`completion` subcommand exception** — the `completion` command overrides root's `PersistentPreRunE` with a no-op to prevent `buildDeps` side effects (config dir creation) during shell completion generation. This is the only permitted exception to the invariant.

**Cobra command grouping** — `cmd.AddGroup(&cobra.Group{ID: "osint", Title: "OSINT Services:"})` then set `GroupID: "osint"` on each `*cobra.Command` struct. Groups render in registration order: `"osint"` → `"aliases"` (conditional) → `"utility"`. Cobra appends groups to the end — to insert a conditional group in the middle, register it positionally inside the constructor, not from `Execute()`. Currently: `"osint"` for OSINT services, `"aliases"` for user-defined aliases (registered inside `newRootCmd` when `len(aliases) > 0`), `"utility"` for completion/version. `newRootCmd(aliases map[string]string)` accepts the alias map; `NewRootCmd()` (exported, for doc gen) passes `nil`. Requires cobra v1.7+; project uses v1.10.2. `cmd.SetHelpCommandGroupID("utility")` — call after `AddCommand(...)` to assign Cobra's built-in `help` subcommand to a named group; without it, `help` appears under a separate "Additional Commands:" section.

**`cmd.MarkFlagsMutuallyExclusive`** — works for persistent flags on root command (calls `mergePersistentFlags()` internally). Used for `"defang"` / `"no-defang"`; the `buildDeps` check remains as a defensive fallback.

**Shell completion generation** — `GenBashCompletionV2(w, true)`, `GenZshCompletion(w)`, `GenFishCompletion(w, true)`, `GenPowerShellCompletionWithDesc(w)` all return `error` — always propagate. Completion subcommand lives in `internal/cli/completion.go`.

**Version build variables** — `var Version`, `Commit`, `Date` in `internal/version/version.go`; injected at build time via `-X github.com/tbckr/trident/internal/version.Version=v1.0.0`. `internal/cli/version.go` imports `internal/version` and reads these vars. `cmd.Version = version.Version` enables `trident --version`; `cmd.SetVersionTemplate("trident {{.Version}}\n")` controls its format.

**Flag completions** — registered inline in `newRootCmd` via two `cmd.RegisterFlagCompletionFunc` calls (for `"output"` and `"pap"`). `RegisterFlagCompletionFunc` returns `error` — discard with `_ =`.

**Cobra positional arg completions** — set `ValidArgsFunction` on the `*cobra.Command` struct for positional argument tab-completion. `RegisterFlagCompletionFunc` is only for named flag completions, not positional args.

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

**PAP level ordering** — ascending activity: `RED(0) < AMBER(1) < GREEN(2) < WHITE(3)`. RED = non-detectable (offline/local); AMBER = 3rd-party APIs; GREEN = direct target interaction; WHITE = unrestricted. `Allows(limit, service Level) bool { return service <= limit }` — a service is blocked when its level exceeds the user's limit. Default `--pap-limit=white` permits everything.

**`pap.MustParse`** — like `Parse` but panics on invalid input; safe to call in subcommands after `buildDeps` has already validated `cfg.PAPLimit`. Subcommands use `pap.MustParse(d.cfg.PAPLimit)` wherever a `pap.Level` is needed at call time.

**`output.ResolveDefang`** — defanging decision function in `internal/output/defang.go`; extracted from CLI so it can be unit-tested (since `internal/cli/` has intentional 0% coverage). Accepts `(papLevel, format, explicitDefang, noDefang)` and encodes all PAP-trigger rules.

**`DefangURL` host extraction** — never use `strings.Index(s, "/")` to find the host/path boundary; it hits the first `/` inside `://`. Find `://` first, skip 3 bytes, then search for the next `/` in `s[hostStart:]`.

**`gofmt` alignment** — never pad struct field types/tags **or map key→value pairs** with extra spaces for visual alignment (e.g., `Input      string` or `"key":    value`); gofmt normalizes both to single-tab separation and `golangci-lint` will fail.

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

### Service Implementations

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
- **Flag→viper key discrepancies** (hyphen→underscore): `--user-agent`→`user_agent`, `--pap-limit`→`pap_limit`, `--no-defang`→`no_defang`. These drive mapstructure tags and env vars (`TRIDENT_USER_AGENT`, `TRIDENT_PAP_LIMIT`, `TRIDENT_NO_DEFANG`).
- **`Config.ConfigFile`** has no mapstructure tag — set manually after `v.Unmarshal(&cfg)` (meta-field, not a viper key).
- **`Config.Aliases`** — `map[string]string` with `mapstructure:"alias"`; file-only, no flag/env binding. Populated by Viper from the `alias:` YAML key.
- **`config.DefaultConfigPath()`** — returns the OS-appropriate default config path without creating the file (unlike `Load`). Use in `Execute()` for pre-parse alias loading.
- **`config.LoadAliases(path)`** — reads only the `alias:` section via a fresh Viper instance; returns empty non-nil map when file missing or key absent.

### Global Flags

| Flag | Default |
|------|---------|
| `--config` | `~/.config/trident/config.yaml` |
| `--verbose` / `-v` | Info level logging |
| `--output` / `-o` | `text` (also: `json`, `plain`) |
| `--concurrency` / `-c` | `10` |
| `--proxy` | — (supports `http://`, `https://`, `socks5://`) |
| `--user-agent` | rotating browser UAs |
| `--pap-limit` | `white` |
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
- **GoReleaser v2 `format_overrides`** — use `formats: [zip]` (list), not `format: zip` (deprecated scalar since v2.6); `goreleaser check` catches this at validation time.
- **GoReleaser nfpms `builds` vs `ids`** — to scope an nfpm to specific build IDs, use `ids: [trident]` (not `builds:`; deprecated since v2.8).
- **GoReleaser nfpms `mtime`** — `file_info.mtime: "{{ .CommitDate }}"` template is supported since v2.6. Use on all `contents` entries for reproducible packages.
- **GoReleaser `before` hooks vs `dist/`** — pipeline order: `dist.CleanPipe` → `before.Pipe` → `dist.Pipe` (emptiness check). Hooks that write to `dist/` cause `dist.Pipe` to fail even with `--clean`. Write hook output to `.build/` instead; reference `.build/` in `archives.files` and `nfpms.contents`.
- **nfpm glob directory collision** — nfpm flattens globbed directories to a single destination, causing content collisions when multiple files share the same basename (e.g., hundreds of transitive `LICENSE` files). Third-party license trees belong in archives only, not `nfpms.contents`.
- **GoReleaser archives `builds_info`** — sets `owner`/`group`/`mtime` on the binary inside archives. Use `mtime: "{{ .CommitDate }}"` for reproducibility (supported since v2.6).
- **GoReleaser archives `files` format** — plain strings (`- LICENSE`) work for bare includes; use object form (`- src: LICENSE\n  info: {owner: root, group: root}`) when per-file ownership is needed. Switch the whole list to object form when any entry needs `info:`.
- **GoReleaser archives `name_template`** — use `title .Os` + arch map for conventional human-readable names: `{{- title .Os }}_{{- if eq .Arch "amd64" }}x86_64{{- else if eq .Arch "386" }}i386{{- else }}{{ .Arch }}{{ end }}{{- if .Arm }}v{{ .Arm }}{{ end -}}`.
- **`cobra/doc` subpackage** — `github.com/spf13/cobra/doc` is within the existing cobra module; no new `go get` needed, but `go mod tidy` will pull transitive deps (`go-md2man`, `blackfriday`).
- **golangci-lint v2 config structure:** `linters-settings` → `linters.settings`; `formatters-settings` → `formatters.settings`; `issues.exclude-rules` → `linters.exclusions.rules`. `goimports.local-prefixes` is an array (not a string). `gosimple` is merged into `staticcheck` — do not list it separately.
- **gosec suppressions:** `gosec.excludes` under `linters.settings` is unreliable; prefer `linters.exclusions.rules` with `text: "G304"` or an inline `//nolint:gosec // reason` comment. `nolintlint` will error if the directive is present but gosec doesn't fire on that line — remove unused nolint directives rather than suppressing them.
- **gosec G304 scope** — `os.ReadFile` with a variable path does **not** trigger G304; do not add a nolint directive there. G304 fires on `os.Open`, `os.OpenFile`, and similar — not `ReadFile`.
- **`strings.CutPrefix`** — golangci-lint's `stringscutprefix` rule fires on `strings.HasPrefix(s, p)` + `strings.TrimPrefix(s, p)` combos; always use `if v, ok := strings.CutPrefix(s, p); ok { ... }` instead.
- **revive `package-comments`:** Every package must have a `// Package foo ...` comment in `doc.go` (never inline in an implementation file). New packages without this will fail lint.
- **cosign v3 signing** — `cosign-installer@v4.x` is required for cosign v3.x (`@v3.x` only installs v2). In GoReleaser `signs:`, use `signature: "${artifact}.sigstore.json"` + `--bundle=${signature}` (v3 replaced `--output-certificate`/`--output-signature` with a single bundle). Do not pin `cosign-release:` in the action — let the installer default handle the version.

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
- `ci.yml` — push/PR: test (with `go mod verify` + tidy check), lint, govulncheck (plain `run` step — no sandbox), license-check (`go-licenses check` against allowlist)
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

**`go-licenses v2`** — invoked via `go run github.com/google/go-licenses/v2@latest`. `--ignore` is a persistent root flag (not shown in `save --help`); always pass `--ignore=github.com/tbckr/trident` to `save` to prevent copying the project's own module. Allowlist: `MIT,Apache-2.0,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0,GPL-3.0,GPL-3.0-only`.

