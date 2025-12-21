# Implementation Plan: Trident

**Project**: trident
**Generated**: 2025-12-21T10:47:00+01:00

## Technical Context & Standards

*Detected Stack & Patterns*
- **Framework**: Go 1.21+ (Greenfield project - no existing code)
- **CLI**: spf13/cobra (no global commands, constructor pattern)
- **Config**: spf13/viper (YAML + env vars)
- **HTTP Client**: imroc/req v3 (no external SDKs)
- **Logging**: log/slog (stdlib only, dynamic LevelVar pattern)
- **Tables**: olekukonko/tablewriter
- **Testing**: stretchr/testify + jarcoal/httpmock
- **Conventions**: 
  - Dependency injection (no global state)
  - run() pattern in main
  - Constructor injection for all services
  - 80% minimum test coverage
  - Black-box testing (package_test)

---

## Phase 1: Project Foundation

- [ ] **Initialize Go Module and Project Structure** (ref: PRD §4.1)
  Task ID: phase-1-foundation-01
  > **Implementation**: Create module structure at project root.
  > **Details**:
  > - Run `go mod init github.com/tbckr/trident`
  > - Create directory structure:
  >   ```
  >   cmd/trident/main.go          # Entry point with run() pattern
  >   internal/
  >     config/config.go           # Viper configuration
  >     output/                    # Output formatters (table, json, plain)
  >     http/client.go             # HTTP client wrapper with interfaces
  >     services/                  # Service implementations
  >       dns/dns.go
  >       asn/asn.go
  >       crtsh/crtsh.go
  >       threatminer/threatminer.go
  >       pgp/pgp.go
  >     ratelimit/limiter.go       # Token bucket rate limiter
  >     opsec/                     # OpSec utilities
  >       pap/pap.go               # PAP level enforcement
  >       defang/defang.go         # Output defanging
  >       useragent/useragent.go   # UA rotation
  >   pkg/                         # Public interfaces (if needed)
  >   ```
  > - Create .gitignore for Go projects
  > - Ensure cmd/trident/main.go uses the run() pattern from AGENTS.md

- [ ] **Implement main.go with run() Pattern** (ref: PRD §4.4, AGENTS.md)
  Task ID: phase-1-foundation-02
  > **Implementation**: Create `cmd/trident/main.go`
  > **Details**:
  > - Implement main() that:
  >   1. Creates context.Background()
  >   2. Initializes slog.LevelVar for dynamic log level
  >   3. Creates JSON logger with LevelVar
  >   4. Calls run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr, logger, programLevel)
  >   5. Exits with error code if run() fails
  > - Implement run() signature:
  >   ```go
  >   func run(ctx context.Context, args []string, getenv func(string) string, 
  >            stdin io.Reader, stdout, stderr io.Writer, 
  >            logger *slog.Logger, levelVar *slog.LevelVar) error
  >   ```
  > - Inside run(): Handle signal.NotifyContext for graceful shutdown
  > - No global variables permitted

- [ ] **Create Root Command Constructor** (ref: PRD §4.2, AGENTS.md)
  Task ID: phase-1-foundation-03
  > **Implementation**: Create `internal/cmd/root.go`
  > **Details**:
  > - Define RootOptions struct to hold all root-level flags
  > - Implement `NewRootCmd(logger *slog.Logger, levelVar *slog.LevelVar, getenv func(string) string) *cobra.Command`
  > - NO global command variables (violates AGENTS.md)
  > - Bind flags in constructor using RootOptions struct pattern (inspired by GoReleaser)
  > - Global flags to implement:
  >   - --config (string, default: ~/.config/trident/config.yaml)
  >   - --verbose/-v (bool, triggers levelVar.Set(slog.LevelDebug))
  >   - --output/-o (string, choices: text|json|plain, default: text)
  >   - --proxy (string, URL)
  >   - --user-agent (string)
  >   - --pap-limit (string, choices: red|amber|green|white, default: white)
  >   - --defang (bool)
  >   - --no-defang (bool)
  >   - --concurrency/-c (int, default: 10)
  > - PersistentPreRunE: Check --verbose flag and call levelVar.Set(slog.LevelDebug)
  > - In run(), bind: rootCmd.SetArgs(), SetIn(), SetOut(), SetErr() before ExecuteContext(ctx)

- [ ] **Implement Configuration Management** (ref: PRD §4.1, §7.3)
  Task ID: phase-1-foundation-04
  > **Implementation**: Create `internal/config/config.go`
  > **Details**:
  > - Use Viper to load config from:
  >   1. Config file (--config flag or default path)
  >   2. Environment variables (prefix: TRIDENT_)
  > - Implement function: `LoadConfig(configPath string, getenv func(string) string) (*Config, error)`
  > - Config struct should include:
  >   ```go
  >   type Config struct {
  >       Proxy      string
  >       UserAgent  string
  >       PAPLimit   string
  >       Concurrency int
  >       // Future: API keys for services
  >   }
  >   ```
  > - Ensure config file has 0600 permissions when created
  > - Environment variable precedence over file values
  > - Never log/print secrets (use masking if debug dump needed)
  > - Use filepath.Join for OS-agnostic paths
  > - Respect XDG_CONFIG_HOME on Linux, AppData on Windows

---

## Phase 2: Core Infrastructure

- [ ] **Implement HTTP Client Wrapper with Interface** (ref: PRD §4.4)
  Task ID: phase-2-infrastructure-01
  > **Implementation**: Create `internal/http/client.go`
  > **Details**:
  > - Define interface for HTTP operations:
  >   ```go
  >   type Client interface {
  >       Get(ctx context.Context, url string) (*Response, error)
  >       Post(ctx context.Context, url string, body io.Reader) (*Response, error)
  >   }
  >   ```
  > - Implement concrete type using imroc/req v3
  > - Constructor: `NewClient(logger *slog.Logger, config *config.Config) Client`
  > - Apply proxy settings from config
  > - Apply User-Agent from config
  > - Enforce HTTPS only (reject HTTP unless explicitly allowed)
  > - Enable TLS verification by default
  > - Support context cancellation
  > - Interface enables mocking with httpmock in tests

- [ ] **Implement Token Bucket Rate Limiter** (ref: PRD §6.2)
  Task ID: phase-2-infrastructure-02
  > **Implementation**: Create `internal/ratelimit/limiter.go`
  > **Details**:
  > - Use golang.org/x/time/rate package
  > - Implement per-service rate limiters
  > - Constructor: `NewLimiter(rps float64, burst int, logger *slog.Logger) *Limiter`
  > - Add jitter (±20% variation) for traffic analysis protection
  > - Respect standard HTTP headers:
  >   - X-RateLimit-Remaining
  >   - Retry-After
  >   - X-RateLimit-Reset
  > - Handle HTTP 429 with exponential backoff
  > - Method: `Wait(ctx context.Context) error`

- [ ] **Implement Worker Pool for Concurrency Control** (ref: PRD §6.1)
  Task ID: phase-2-infrastructure-03
  > **Implementation**: Create `internal/worker/pool.go`
  > **Details**:
  > - Implement worker pool pattern using bounded channels/semaphores
  > - Pool size determined by --concurrency flag
  > - Constructor: `NewPool(size int, logger *slog.Logger) *Pool`
  > - Method: `Process(ctx context.Context, inputs <-chan Input, fn ProcessFunc) <-chan Result`
  > - Prevent resource exhaustion and accidental DoS
  > - Graceful shutdown on context cancellation
  > - Error aggregation from workers

- [ ] **Implement Input Handler (Stdin + Args)** (ref: PRD §4.3)
  Task ID: phase-2-infrastructure-04
  > **Implementation**: Create `internal/input/reader.go`
  > **Details**:
  > - Function: `GetInputs(args []string, stdin io.Reader) ([]string, error)`
  > - Priority logic:
  >   1. If args provided -> return args
  >   2. If no args -> check stdin (pipe detection)
  >   3. Process stdin line-by-line
  > - Trim whitespace and ignore empty lines
  > - Validate each input before processing
  > - Example usage: `cat domains.txt | trident dns`

---

## Phase 3: Output System

- [ ] **Implement Output Formatters** (ref: PRD §6.4)
  Task ID: phase-3-output-01
  > **Implementation**: Create `internal/output/formatter.go`
  > **Details**:
  > - Define interface:
  >   ```go
  >   type Formatter interface {
  >       Format(data interface{}) (string, error)
  >   }
  >   ```
  > - Implement three formatters:
  >   - TableFormatter (using olekukonko/tablewriter)
  >   - JSONFormatter (using encoding/json)
  >   - PlainFormatter (raw list, one per line)
  > - Factory function: `NewFormatter(format string, defang bool) Formatter`
  > - All formatters must support defanging when enabled
  > - Sanitize ANSI escape sequences to prevent terminal injection

- [ ] **Implement Defanging Logic** (ref: PRD §8.5)
  Task ID: phase-3-output-02
  > **Implementation**: Create `internal/opsec/defang/defang.go`
  > **Details**:
  > - Function: `Defang(input string, defangEnabled bool) string`
  > - Transformations:
  >   - Domains: example.com -> example[.]com
  >   - IPs: 1.2.3.4 -> 1.2.3[.]4
  >   - URLs: https://malware.com -> hxxps://malware[.]com
  > - Triggered when:
  >   1. --pap-limit is AMBER or RED (automatic)
  >   2. --defang is set (explicit)
  >   3. NOT when --no-defang is set (override)
  > - JSON remains raw unless --defang explicitly set
  > - Plain mode follows same rules as text

---

## Phase 4: OpSec & Security

- [ ] **Implement PAP Level System** (ref: PRD §8.4)
  Task ID: phase-4-opsec-01
  > **Implementation**: Create `internal/opsec/pap/pap.go`
  > **Details**:
  > - Define PAP levels as enum/const:
  >   ```go
  >   type Level int
  >   const (
  >       Red Level = iota    // Offline/Local
  >       Amber               // 3rd Party APIs
  >       Green               // Direct interaction
  >       White               // Unrestricted
  >   )
  >   ```
  > - Function: `Enforce(commandLevel, userLimit Level) error`
  > - Return error if commandLevel > userLimit
  > - Each service must declare its PAP level
  > - Display PAP level in CLI help and execution logs

- [ ] **Implement User-Agent Rotation** (ref: PRD §8.2)
  Task ID: phase-4-opsec-02
  > **Implementation**: Create `internal/opsec/useragent/useragent.go`
  > **Details**:
  > - Maintain list of modern browser User-Agent strings:
  >   - Chrome on Windows
  >   - Firefox on Linux
  >   - Safari on macOS
  > - Function: `GetRandomUA() string`
  > - Do NOT use "Trident" or "Go-http-client" by default
  > - Allow override via --user-agent flag
  > - Rotate randomly on each request or session

- [ ] **Implement Proxy Support with DNS Leak Prevention** (ref: PRD §8.1)
  Task ID: phase-4-opsec-03
  > **Implementation**: Modify `internal/http/client.go`
  > **Details**:
  > - Support proxy types: HTTP, HTTPS, SOCKS5
  > - For SOCKS5: Ensure DNS resolution happens through proxy (remote DNS)
  > - Use net.Dialer with proxy configuration
  > - Validate proxy URL format
  > - Log proxy usage at debug level
  > - Ensure all HTTP/HTTPS traffic routes through proxy

- [ ] **Implement Input Validation & Sanitization** (ref: PRD §7.1)
  Task ID: phase-4-opsec-04
  > **Implementation**: Create `internal/validation/validator.go`
  > **Details**:
  > - Strict regex validation for:
  >   - Domains: RFC-compliant hostnames
  >   - IPs: net.ParseIP for IPv4/IPv6
  >   - ASNs: ^AS\d+$ pattern
  >   - Hashes: MD5=32 hex, SHA1=40 hex, SHA256=64 hex
  > - Prevent command injection
  > - Prevent path traversal
  > - Function signature: `ValidateDomain(input string) error`
  > - Use before any external API call

---

## Phase 5: Service Implementation (DNS)

- [ ] **Implement DNS Service** (ref: PRD §5.1.1)
  Task ID: phase-5-dns-01
  > **Implementation**: Create `internal/services/dns/dns.go`
  > **Details**:
  > - PAP Level: GREEN
  > - Use Go's net package (no HTTP required)
  > - Constructor: `NewService(logger *slog.Logger, resolver *net.Resolver) *Service`
  > - Function: `Lookup(ctx context.Context, target string) (*DNSResult, error)`
  > - Query types: A, AAAA, MX, NS, TXT, CNAME
  > - Return struct with all record types
  > - Handle timeouts via context
  > - Validate input is domain or IP

- [ ] **Create DNS Command** (ref: PRD §5.1.1)
  Task ID: phase-5-dns-02
  > **Implementation**: Create `internal/cmd/dns.go`
  > **Details**:
  > - Constructor: `NewDNSCmd(logger *slog.Logger, config *config.Config) *cobra.Command`
  > - Use: `trident dns [domain or IP]`
  > - Support stdin bulk input
  > - Respect --concurrency with worker pool
  > - Respect --pap-limit (check against GREEN)
  > - Output formatters: table (default), json, plain
  > - Example: `cat domains.txt | trident dns --output json`

- [ ] **Write Tests for DNS Service** (ref: PRD §4.4)
  Task ID: phase-5-dns-03
  > **Implementation**: Create `internal/services/dns/dns_test.go`
  > **Details**:
  > - Package: `dns_test` (black-box testing)
  > - Use testify for assertions
  > - Table-driven tests with t.Run()
  > - Mock net.Resolver with custom lookup functions
  > - Test cases:
  >   - Valid domain lookup
  >   - Valid IP reverse lookup
  >   - Invalid input handling
  >   - Timeout scenarios
  >   - Context cancellation
  > - Achieve >80% coverage

---

## Phase 6: Service Implementation (ASN)

- [ ] **Implement ASN Service** (ref: PRD §5.1.2)
  Task ID: phase-6-asn-01
  > **Implementation**: Create `internal/services/asn/asn.go`
  > **Details**:
  > - PAP Level: AMBER
  > - Use Team Cymru DNS TXT records
  > - Query format: `<reversed-ip>.origin.asn.cymru.com`
  > - For ASN: `AS<number>.asn.cymru.com`
  > - Constructor: `NewService(logger *slog.Logger, resolver *net.Resolver) *Service`
  > - Function: `Lookup(ctx context.Context, target string) (*ASNResult, error)`
  > - Parse TXT response for: ASN, Description, Country, Registry, Prefixes
  > - Do NOT use os/exec to call dig
  > - Use net.Resolver.LookupTXT()

- [ ] **Create ASN Command**
  Task ID: phase-6-asn-02
  > **Implementation**: Create `internal/cmd/asn.go`
  > **Details**:
  > - Constructor pattern (no globals)
  > - Use: `trident asn [AS number or IP]`
  > - Support stdin bulk input
  > - Respect --concurrency
  > - Respect --pap-limit (check against AMBER)
  > - Apply rate limiting for Team Cymru
  > - Output formatters: table, json, plain

- [ ] **Write Tests for ASN Service**
  Task ID: phase-6-asn-03
  > **Implementation**: Create `internal/services/asn/asn_test.go`
  > **Details**:
  > - Black-box testing (asn_test package)
  > - Mock DNS resolver responses
  > - Test TXT record parsing
  > - Test both IP and ASN input
  > - Test invalid inputs
  > - Achieve >80% coverage

---

## Phase 7: Service Implementation (Crt.sh)

- [ ] **Implement Crt.sh Service** (ref: PRD §5.1.3)
  Task ID: phase-7-crtsh-01
  > **Implementation**: Create `internal/services/crtsh/crtsh.go`
  > **Details**:
  > - PAP Level: AMBER
  > - Endpoint: `https://crt.sh/?q=%.{domain}&output=json`
  > - Constructor: `NewService(client http.Client, logger *slog.Logger, limiter *ratelimit.Limiter) *Service`
  > - Function: `Search(ctx context.Context, domain string) (*CrtshResult, error)`
  > - Parse JSON response for: subdomains, cert dates, CAs
  > - No external SDK - use imroc/req directly
  > - Apply rate limiting with jitter
  > - Handle HTTP errors gracefully

- [ ] **Create Crt.sh Command**
  Task ID: phase-7-crtsh-02
  > **Implementation**: Create `internal/cmd/crtsh.go`
  > **Details**:
  > - Constructor pattern
  > - Use: `trident crtsh [domain]`
  > - Support stdin bulk input
  > - Respect --concurrency
  > - Respect --pap-limit (AMBER)
  > - Output formatters: table, json, plain

- [ ] **Write Tests for Crt.sh Service**
  Task ID: phase-7-crtsh-03
  > **Implementation**: Create `internal/services/crtsh/crtsh_test.go`
  > **Details**:
  > - Black-box testing (crtsh_test package)
  > - Use httpmock to mock HTTP responses
  > - Recorded fixtures for responses
  > - Test JSON parsing
  > - Test error handling (404, timeout)
  > - Achieve >80% coverage

---

## Phase 8: Service Implementation (ThreatMiner)

- [ ] **Implement ThreatMiner Service** (ref: PRD §5.1.4)
  Task ID: phase-8-threatminer-01
  > **Implementation**: Create `internal/services/threatminer/threatminer.go`
  > **Details**:
  > - PAP Level: AMBER
  > - Base URL: `https://api.threatminer.org/v2/`
  > - Endpoints:
  >   - Domain: `/domain.php?q={domain}&rt={report_type}`
  >   - IP: `/host.php?q={ip}&rt={report_type}`
  >   - Hash: `/sample.php?q={hash}&rt={report_type}`
  > - Constructor: `NewService(client http.Client, logger *slog.Logger, limiter *ratelimit.Limiter) *Service`
  > - Functions for: `LookupDomain()`, `LookupIP()`, `LookupHash()`
  > - Parse JSON for: Passive DNS, Malware hashes, WHOIS
  > - No external SDK

- [ ] **Create ThreatMiner Command**
  Task ID: phase-8-threatminer-02
  > **Implementation**: Create `internal/cmd/threatminer.go`
  > **Details**:
  > - Constructor pattern
  > - Use: `trident threatminer [domain|ip|hash]`
  > - Auto-detect input type (domain vs IP vs hash)
  > - Support stdin bulk input
  > - Respect --pap-limit (AMBER)
  > - Output formatters: table, json, plain

- [ ] **Write Tests for ThreatMiner Service**
  Task ID: phase-8-threatminer-03
  > **Implementation**: Create `internal/services/threatminer/threatminer_test.go`
  > **Details**:
  > - Black-box testing
  > - Mock HTTP responses with httpmock
  > - Test all endpoint types
  > - Test auto-detection logic
  > - Achieve >80% coverage

---

## Phase 9: Service Implementation (PGP)

- [ ] **Implement PGP Key Search Service** (ref: PRD §5.1.5)
  Task ID: phase-9-pgp-01
  > **Implementation**: Create `internal/services/pgp/pgp.go`
  > **Details**:
  > - PAP Level: AMBER
  > - Use HKP protocol (keys.openpgp.org)
  > - Endpoint: `https://keys.openpgp.org/vks/v1/by-email/{email}`
  > - Constructor: `NewService(client http.Client, logger *slog.Logger, limiter *ratelimit.Limiter) *Service`
  > - Function: `Search(ctx context.Context, email string) (*PGPResult, error)`
  > - Parse response for: Key ID, Fingerprint, User IDs, Creation Date
  > - Support both email and name search
  > - Prioritize HKPS (HTTPS) over HKP (HTTP)

- [ ] **Create PGP Command**
  Task ID: phase-9-pgp-02
  > **Implementation**: Create `internal/cmd/pgp.go`
  > **Details**:
  > - Constructor pattern
  > - Use: `trident pgp [email or name]`
  > - Support stdin bulk input
  > - Respect --pap-limit (AMBER)
  > - Output formatters: table, json, plain

- [ ] **Write Tests for PGP Service**
  Task ID: phase-9-pgp-03
  > **Implementation**: Create `internal/services/pgp/pgp_test.go`
  > **Details**:
  > - Black-box testing
  > - Mock HTTP responses
  > - Test email validation
  > - Test response parsing
  > - Achieve >80% coverage

---

## Phase 10: CI/CD & Tooling

- [ ] **Configure golangci-lint** (ref: PRD §4.5)
  Task ID: phase-10-cicd-01
  > **Implementation**: Create `.golangci.yml`
  > **Details**:
  > - Enable strict linting rules
  > - Fail build on any lint error
  > - Enable linters: gofmt, govet, errcheck, staticcheck, gosec
  > - Set timeout: 5m
  > - Configure per-directory exclusions if needed

- [ ] **Setup GitHub Actions CI Pipeline** (ref: PRD §4.5)
  Task ID: phase-10-cicd-02
  > **Implementation**: Create `.github/workflows/ci.yml`
  > **Details**:
  > - Jobs:
  >   1. **Test**: Run `go test -v -race -coverprofile=coverage.out ./...`
  >   2. **Lint**: Run golangci-lint
  >   3. **Security**: Run gosec for SAST
  >   4. **Vulnerability**: Run govulncheck for SCA
  > - Matrix: Test on Linux, macOS, Windows
  > - Upload coverage reports
  > - Fail if coverage < 80%
  > - Cache Go modules for speed

- [ ] **Configure GoReleaser** (ref: PRD §4.5)
  Task ID: phase-10-cicd-03
  > **Implementation**: Create `.goreleaser.yml`
  > **Details**:
  > - Build configuration:
  >   - Binary name: trident
  >   - Targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
  >   - Flags: -trimpath for reproducible builds
  >   - ldflags: -s -w -X main.version={{.Version}}
  > - Generate SBOM using CycloneDX format
  > - Sign binaries and checksums with Cosign
  > - Upload to GitHub Releases
  > - Create archives (tar.gz, zip)

- [ ] **Setup Renovate for Dependency Updates** (ref: PRD §4.5)
  Task ID: phase-10-cicd-04
  > **Implementation**: Create `renovate.json`
  > **Details**:
  > - Auto-detect Go dependencies
  > - Schedule: Weekly
  > - Auto-merge minor/patch updates after CI passes
  > - Group Go module updates
  > - Create PRs for major updates

- [ ] **Create Release Workflow** (ref: PRD §4.5)
  Task ID: phase-10-cicd-05
  > **Implementation**: Create `.github/workflows/release.yml`
  > **Details**:
  > - Trigger: On git tag push (v*)
  > - Run full test suite
  > - Run security scans
  > - Execute GoReleaser
  > - Upload SBOM and signatures
  > - Follow Semantic Versioning

---

## Phase 11: Documentation & Polish

- [ ] **Write README.md**
  Task ID: phase-11-docs-01
  > **Implementation**: Create `README.md`
  > **Details**:
  > - Project description and goals
  > - Installation instructions
  > - Quick start guide
  > - Command examples for each service
  > - Global flags documentation
  > - OpSec features explanation
  > - PAP levels reference
  > - Configuration file format
  > - Contributing guidelines

- [ ] **Create Example Configuration File**
  Task ID: phase-11-docs-02
  > **Implementation**: Create `config.example.yaml`
  > **Details**:
  > - Document all configuration options
  > - Include comments explaining each field
  > - Show proxy configuration examples
  > - Show PAP limit examples
  > - Note: API keys section for Phase 3

- [ ] **Add CLI Help Documentation**
  Task ID: phase-11-docs-03
  > **Implementation**: Enhance cobra command descriptions
  > **Details**:
  > - Long descriptions for each command
  > - Usage examples in help text
  > - Flag descriptions with defaults
  > - Show PAP level for each command in help

- [ ] **Implement Self-Cleanup Command** (ref: PRD §8.6)
  Task ID: phase-11-docs-04
  > **Implementation**: Create `internal/cmd/burn.go`
  > **Details**:
  > - Command: `trident burn`
  > - Securely delete:
  >   - Config file (~/.config/trident/)
  >   - Logs (if any)
  >   - Cache files
  >   - Binary itself (attempt self-deletion)
  > - Use secure deletion (multiple overwrites)
  > - Warning: Binary self-deletion NOT supported on Windows (log warning instead)
  > - Require confirmation prompt

---

## Phase 12: Integration Testing

- [ ] **Create Integration Test Suite**
  Task ID: phase-12-integration-01
  > **Implementation**: Create `test/integration/`
  > **Details**:
  > - End-to-end CLI tests
  > - Test stdin piping
  > - Test all output formats
  > - Test PAP enforcement
  > - Test defanging
  > - Test concurrency with worker pool
  > - Use recorded fixtures (no real API calls)

- [ ] **Performance Benchmarking**
  Task ID: phase-12-integration-02
  > **Implementation**: Create benchmark tests
  > **Details**:
  > - Benchmark bulk processing with worker pool
  > - Benchmark different concurrency levels
  > - Benchmark output formatters
  > - Use Go's testing.B framework
  > - Document baseline performance

- [ ] **Security Testing**
  Task ID: phase-12-integration-03
  > **Implementation**: Run security audit
  > **Details**:
  > - Verify no secrets in logs
  > - Test input validation against injection
  > - Verify HTTPS enforcement
  > - Test proxy DNS leak prevention
  > - Verify file permissions on config
  > - Run gosec and address findings

---

*Generated by Clavix /clavix-plan*
