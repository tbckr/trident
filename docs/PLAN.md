# Trident Implementation Plan

## Overview

This document outlines the implementation plan for **Trident**, a high-performance OSINT CLI tool written in Go. The plan is divided into multiple phases, each building upon the previous one to deliver incremental value while maintaining code quality, security, and testability.

---

## Phase 0: Foundation & Infrastructure (Week 1-2)

**Goal:** Establish the project foundation, development environment, and CI/CD pipeline.

### 0.1 Project Setup
- [x] Initialize Go module (`go.mod`)
- [x] Configure `.editorconfig` for consistent code formatting
- [x] Set up `.gitignore` for Go projects
- [ ] Create basic directory structure:
  ```
  trident/
  ├── cmd/trident/          # CLI entry point
  ├── internal/             # Private application code
  │   ├── config/          # Configuration management (Viper)
  │   ├── input/           # Stdin/args input handling
  │   ├── opsec/           # OpSec features (PAP, defang)
  │   ├── output/          # Output formatters (table, JSON, plain)
  │   ├── ratelimit/       # Rate limiting implementation
  │   ├── worker/          # Worker pool for concurrency
  │   └── services/        # Service implementations
  ├── pkg/                 # Public/reusable packages
  └── test/                # Integration tests
  ```

### 0.2 Core Dependencies Installation
- [ ] Install and configure:
  - `github.com/spf13/cobra` - CLI framework
  - `github.com/spf13/viper` - Configuration management
  - `github.com/imroc/req/v3` - HTTP client
  - `github.com/olekukonko/tablewriter` - Table output
  - `github.com/stretchr/testify` - Testing framework
  - `github.com/jarcoal/httpmock` - HTTP mocking for tests
  - `golang.org/x/time/rate` - Rate limiting

### 0.3 CI/CD Pipeline (GitHub Actions)
- [ ] Create `.github/workflows/test.yml`:
  - Run `go test -v -race -coverprofile=coverage.txt ./...`
  - Enforce minimum 80% code coverage
  - Upload coverage reports
- [ ] Create `.github/workflows/lint.yml`:
  - Configure `golangci-lint` with strict settings
  - Ensure build fails on any linting error
- [ ] Create `.github/workflows/security.yml`:
  - Run `gosec` for SAST
  - Run `govulncheck` for SCA/vulnerability scanning
- [ ] Configure GoReleaser (`.goreleaser.yml`):
  - Multi-platform builds (Linux, macOS, Windows)
  - SBOM generation using CycloneDX
  - Binary signing with Cosign
  - Archive creation and GitHub release publishing
- [ ] Set up Renovate for dependency management

### 0.4 Documentation
- [x] Create `docs/PRD.md` (Product Requirements Document)
- [x] Create `docs/PLAN.md` (this document)
- [ ] Create `README.md` with project overview and quick start
- [ ] Create `CONTRIBUTING.md` with development guidelines

**Deliverables:**
- ✅ Functional CI/CD pipeline
- ✅ Automated testing, linting, and security scanning
- ✅ GoReleaser configured for releases
- ✅ Development environment ready

---

## Phase 1: Core Framework & MVP Services (Week 3-6)

**Goal:** Implement the CLI framework, core infrastructure, and 5 keyless services (DNS, ASN, Crt.sh, ThreatMiner, PGP).

### 1.1 Configuration Management (`internal/config/`)
- [ ] Implement Viper-based configuration loader
- [ ] Define configuration schema:
  ```yaml
  proxy: ""
  user_agent: ""
  pap_limit: "white"
  concurrency: 10
  output_format: "text"
  defang: false
  services:
    # Future: API keys will go here
  ```
- [ ] Support reading from:
  - Use OS specific path based on stdlib library.
  - Custom path via `--config` flag
  - Environment variables (e.g., `TRIDENT_PROXY`)
- [ ] Implement secure config file creation with proper permissions (0600)
- [ ] Add secret masking for debug output
- [ ] **Tests:** Unit tests for config loading, merging, and validation

### 1.2 CLI Framework (`cmd/trident/`)
- [ ] Set up Cobra root command with global flags:
  - `--config` (config file path)
  - `--verbose, -v` (enable debug logging)
  - `--output, -o` (format: text, json, plain)
  - `--proxy` (proxy URL)
  - `--user-agent` (custom UA string)
  - `--pap-limit` (red, amber, green, white)
  - `--defang` (enable defanging)
  - `--no-defang` (disable defanging)
  - `--concurrency, -c` (worker count)
- [ ] Implement flag binding to Viper configuration
- [ ] Add version command (`trident version`)
- [ ] **Tests:** Command structure and flag parsing tests

### 1.3 Logging (`internal/logging/`)
- [ ] Wrap `log/slog` for structured logging
- [ ] Implement log level switching based on `--verbose` flag
- [ ] Ensure no secrets are logged
- [ ] **Tests:** Log level and masking tests

### 1.4 Input Handling (`internal/input/`)
- [x] Implement dual input mode (args vs. stdin)
- [x] Priority logic:
  1. Process CLI args if provided
  2. Fallback to stdin if available
  3. Error if neither provided
- [x] Line-by-line parsing from stdin
- [x] Trim whitespace and ignore empty lines
- [ ] **Tests:** Mock stdin tests, argument parsing tests

### 1.5 OpSec Features (`internal/opsec/`)

#### 1.5.1 PAP (Permissible Actions Protocol) (`internal/opsec/pap/`)
- [x] Define PAP levels enum: RED, AMBER, GREEN, WHITE
- [x] Implement PAP enforcement logic
- [x] Add PAP metadata to each service
- [ ] Block command execution if service PAP > user limit
- [ ] **Tests:** PAP comparison and enforcement tests

#### 1.5.2 Defanging (`internal/opsec/defang/`)
- [x] Implement defanging for:
  - Domains: `example.com` → `example[.]com`
  - IPs: `1.2.3.4` → `1.2.3[.]4`
  - URLs: `https://example.com` → `hxxps://example[.]com`
- [ ] Implement trigger logic:
  - Auto-enable for PAP ≤ AMBER
  - Enable via `--defang`
  - Override via `--no-defang`
- [ ] Handle format-specific rules (JSON raw unless explicit)
- [ ] **Tests:** Defanging string transformations, trigger logic tests

### 1.6 Output Formatting (`internal/output/`)
- [ ] Implement output formatter interface:
  ```go
  type Formatter interface {
      Format(data interface{}) (string, error)
  }
  ```
- [ ] Implement formatters:
  - **Text:** ASCII table using `tablewriter`
  - **JSON:** Pretty-printed JSON
  - **Plain:** One item per line (for piping)
- [ ] Integrate defanging into formatters
- [ ] **Tests:** Format output for all three modes, verify defanging integration

### 1.7 Rate Limiting (`internal/ratelimit/`)
- [x] Implement token bucket rate limiter using `golang.org/x/time/rate`
- [x] Add per-service rate limit configuration
- [ ] Implement random jitter (±20%) to avoid pattern detection
- [ ] Add HTTP 429 detection and backoff logic using `Retry-After` header
- [ ] Respect `X-RateLimit-*` headers
- [ ] **Tests:** Rate limiter behavior, jitter distribution tests

### 1.8 Concurrency (`internal/worker/`)
- [x] Implement worker pool pattern
- [x] Use semaphore/bounded channel for concurrency control
- [x] Configurable pool size via `--concurrency` flag
- [ ] Graceful shutdown on interrupt (SIGINT/SIGTERM)
- [ ] **Tests:** Worker pool tests with mock tasks

### 1.9 HTTP Client Abstraction (`internal/httpclient/`)
- [ ] Wrap `imroc/req` with interface for testability:
  ```go
  type HTTPClient interface {
      Get(url string, opts ...interface{}) (*Response, error)
      Post(url string, opts ...interface{}) (*Response, error)
  }
  ```
- [ ] Configure default settings:
  - HTTPS enforcement
  - TLS verification enabled
  - Proxy support (HTTP, HTTPS, SOCKS5)
  - Custom User-Agent rotation
- [ ] Implement DNS leak prevention for SOCKS5 proxies
- [ ] **Tests:** Mock HTTP client for unit tests using `httpmock`

### 1.10 User-Agent Management (`internal/useragent/`)
- [ ] Use standard useragents from req/v3
- [ ] Allow override via `--user-agent` flag
- [ ] **Tests:** Rotation logic, override tests

### 1.11 Input Validation (`internal/validation/`)
- [ ] Implement validators for:
  - **Domains:** RFC-compliant hostname validation
  - **IPs:** `net.ParseIP` validation (IPv4/IPv6)
  - **ASNs:** Regex `^AS\d+$`
  - **Hashes:** Length/charset validation (MD5, SHA1, SHA256)
  - **Emails:** Basic email format validation
- [ ] Sanitize input to prevent injection attacks
- [ ] **Tests:** Validator tests for valid/invalid inputs

### 1.12 Service Implementation (`internal/services/`)

**Service Interface:**
```go
type Service interface {
    Name() string
    Execute(ctx context.Context, input string) (interface{}, error)
    PAPLevel() pap.Level
}
```

#### 1.12.1 DNS Service (`dns`)
- [ ] Use Go's native `net` package (no HTTP)
- [ ] Implement queries for: A, AAAA, MX, NS, TXT, CNAME
- [ ] Support both domain and IP (reverse DNS)
- [ ] PAP Level: **GREEN**
- [ ] **Tests:** Mock DNS resolver, test various record types

#### 1.12.2 ASN Service (`asn`)
- [ ] Query Team Cymru via DNS TXT records (`<reversed-ip>.origin.asn.cymru.com`)
- [ ] Use `net.Resolver`, not `os/exec` with `dig`
- [ ] Parse ASN, Description, Country, Registry
- [ ] Support both ASN lookup and IP-to-ASN lookup
- [ ] PAP Level: **AMBER**
- [ ] **Tests:** Mock DNS responses with Cymru format

#### 1.12.3 Crt.sh Service (`crtsh`)
- [ ] HTTP GET to `https://crt.sh/?q=%.{domain}&output=json`
- [ ] Parse JSON response
- [ ] Extract subdomains, issuance dates, CAs
- [ ] Handle rate limiting and errors gracefully
- [ ] PAP Level: **AMBER**
- [ ] **Tests:** Mock HTTP responses with sample crt.sh JSON

#### 1.12.4 ThreatMiner Service (`threatminer`)
- [ ] Implement endpoints:
  - Domain: `https://api.threatminer.org/v2/domain.php?q={domain}&rt=2`
  - IP: `https://api.threatminer.org/v2/host.php?q={ip}&rt=2`
  - Hash: `https://api.threatminer.org/v2/sample.php?q={hash}&rt=1`
- [ ] Parse JSON responses for passive DNS, Whois, malware associations
- [ ] PAP Level: **AMBER**
- [ ] **Tests:** Mock ThreatMiner API responses

#### 1.12.5 PGP Service (`pgp`)
- [ ] Query HKP servers (default: `https://keys.openpgp.org`)
- [ ] Search by email: `/pks/lookup?search={email}&op=index`
- [ ] Parse response for Key ID, Fingerprint, User IDs, Creation Date
- [ ] Handle missing keys gracefully
- [ ] PAP Level: **AMBER**
- [ ] **Tests:** Mock HKP server responses

### 1.13 Service Commands (`cmd/trident/commands/`)
- [ ] Create Cobra commands for each service:
  - `trident dns <domain|ip>`
  - `trident asn <asn|ip>`
  - `trident crtsh <domain>`
  - `trident threatminer <domain|ip|hash>`
  - `trident pgp <email>`
- [ ] Integrate input handling (args + stdin)
- [ ] Apply worker pool for bulk processing
- [ ] Apply rate limiting
- [ ] Apply PAP enforcement
- [ ] Wire output formatters
- [ ] **Tests:** End-to-end command tests with mocked services

### 1.14 Security Hardening
- [ ] Implement ANSI escape sequence sanitization for terminal output
- [ ] Enforce HTTPS-only for all HTTP services
- [ ] Add `--insecure` flag (disabled by default) with warnings
- [ ] Validate all inputs before API calls
- [ ] **Tests:** Injection attack prevention tests

### 1.15 Integration Testing
- [ ] Create integration tests for each service with recorded fixtures
- [ ] Test stdin processing with bulk inputs
- [ ] Test concurrency edge cases (no race conditions)
- [ ] Test PAP enforcement across all services
- [ ] Test defanging in all output formats
- [ ] **Coverage Goal:** Achieve minimum 80% code coverage

**Phase 1 Deliverables:**
- ✅ Working CLI with 5 keyless services (DNS, ASN, Crt.sh, ThreatMiner, PGP)
- ✅ Full OpSec feature support (PAP, defanging, proxy, UA rotation)
- ✅ Robust input handling (args + stdin)
- ✅ Three output formats (text, JSON, plain)
- ✅ 80%+ test coverage
- ✅ CI/CD pipeline passing all checks
- ✅ First release (v0.1.0) via GoReleaser

---

## Phase 2: Caching, Offline Databases & Additional Passive Services (Week 7-10)

**Goal:** Enhance functionality with caching, offline data, and additional keyless services.

### 2.1 Caching Layer (`internal/cache/`)
- [ ] Implement TTL-based in-memory cache (e.g., using `patrickmn/go-cache`)
- [ ] Persist cache to disk (optional, via config)
- [ ] Add `--no-cache` flag to bypass
- [ ] **Tests:** Cache hit/miss, expiration tests

### 2.2 Offline Databases
- [ ] Integrate GeoIP database (e.g., MaxMind GeoLite2)
  - Store in `~/.config/trident/data/`
  - Provide update command: `trident update-db --geoip`
- [ ] Implement Umbrella Top 1M domain list check (CSV-based, offline)
  - PAP Level: **RED** (local lookup)
- [ ] **Tests:** Database loading and query tests

### 2.3 Additional Passive Services

#### 2.3.1 Cache Service (`cache`)
- [ ] Query Archive.org Wayback Machine
- [ ] Query Google Cache (if available)
- [ ] PAP Level: **AMBER**
- [ ] **Tests:** Mock HTTP responses

#### 2.3.2 Quad9 Service (`quad9`)
- [ ] DNS query to Quad9 blocked domain check
- [ ] PAP Level: **AMBER**
- [ ] **Tests:** Mock DNS responses

#### 2.3.3 Tor Service (`tor`)
- [ ] Download and parse Tor exit node list
- [ ] Check if IP is a Tor exit node
- [ ] PAP Level: **AMBER**
- [ ] **Tests:** Mock exit node list parsing

#### 2.3.4 Robtex Service (`robtex`)
- [ ] Implement limited/scraping-based Robtex queries
- [ ] PAP Level: **AMBER**
- [ ] **Tests:** Mock HTTP responses

### 2.4 Performance Optimization
- [ ] Profile application using `pprof`
- [ ] Optimize hot paths identified
- [ ] Benchmark worker pool performance
- [ ] **Tests:** Benchmark tests for critical paths

### 2.5 Documentation
- [ ] User guide for all Phase 1 + 2 services
- [ ] Configuration examples
- [ ] OpSec best practices guide
- [ ] **Deliverable:** `docs/USER_GUIDE.md`

**Phase 2 Deliverables:**
- ✅ Caching layer for improved performance
- ✅ Offline databases (GeoIP, Umbrella Top 1M)
- ✅ 4 additional passive services
- ✅ Comprehensive user documentation
- ✅ Release v0.2.0

---

## Phase 3: API-Key Services & Advanced OpSec (Week 11-16)

**Goal:** Integrate API-key dependent services and advanced operational security features.

### 3.1 API Key Management
- [ ] Extend configuration schema for API keys:
  ```yaml
  services:
    shodan:
      api_key: ""
    virustotal:
      api_key: ""
    # ... other services
  ```
- [ ] Support environment variable injection (e.g., `TRIDENT_SHODAN_API_KEY`)
- [ ] Validate API keys before execution
- [ ] **Tests:** Config loading with secrets, env var precedence

### 3.2 API-Key Dependent Services (40+ services)
Implement services from the PRD Appendix, including:
- [ ] **Shodan** - Network intelligence
- [ ] **VirusTotal** - Malware/threat intelligence
- [ ] **Censys** - Internet-wide scanning data
- [ ] **SecurityTrails** - DNS history
- [ ] **AlienVault OTX** - Threat intelligence
- [ ] **Have I Been Pwned** - Breach data
- [ ] **GreyNoise** - Internet noise analysis
- [ ] **Hunter.io** - Email finder
- [ ] **URLScan.io** - URL scanning (PAP: **GREEN**)
- [ ] ... (see PRD Appendix for full list)

**Implementation Strategy:**
- Create generic service factory pattern
- Implement one service at a time with tests
- Prioritize by popularity/usefulness
- Each service requires 80% test coverage

### 3.3 Advanced OpSec Features

#### 3.3.1 TLS Fingerprinting Evasion
- [ ] Integrate `github.com/refraction-networking/utls`
- [ ] Mimic Chrome/Firefox TLS handshakes (JA3/JA4 evasion)
- [ ] Configurable via flag or config
- [ ] **Tests:** Verify TLS ClientHello matches browser profiles

#### 3.3.2 Honeypot Detection
- [ ] Implement passive canary checks before active scans
- [ ] Detect common honeypot signatures
- [ ] Warn user or abort based on config
- [ ] **Tests:** Mock honeypot responses

#### 3.3.3 Core Dump Prevention
- [ ] Implement memory protection on Linux/macOS:
  - Use `prctl(PR_SET_DUMPABLE, 0)` on Linux
  - Use `ptrace(PT_DENY_ATTACH, 0)` on macOS
- [ ] Log warning on Windows (not supported in Phase 3)
- [ ] **Tests:** Platform-specific tests

#### 3.3.4 Encrypted Workspace (US-12)
- [ ] Implement session passphrase on startup
- [ ] Encrypt all artifacts using AES-GCM or ChaCha20-Poly1305:
  - Logs
  - Cache
  - Output files
- [ ] Provide decryption utility
- [ ] **Tests:** Encryption/decryption round-trip tests

### 3.4 Enhanced Rate Limiting
- [ ] Per-service configurable rate limits
- [ ] Adaptive rate limiting based on `X-RateLimit-*` headers
- [ ] Exponential backoff for 429 responses
- [ ] **Tests:** Service-specific rate limit tests

### 3.5 Advanced Output Features
- [ ] Implement output templates (e.g., custom Go templates)
- [ ] Add CSV output format
- [ ] Add XML output format (optional)
- [ ] **Tests:** Format tests for new outputs

**Phase 3 Deliverables:**
- ✅ 40+ API-key services integrated
- ✅ TLS fingerprinting evasion
- ✅ Honeypot detection
- ✅ Encrypted workspace mode
- ✅ Core dump prevention (Linux/macOS)
- ✅ Release v1.0.0 (major milestone)

---

## Phase 4: Behavioral Mimicry & VEX (Week 17-20)

**Goal:** Implement advanced evasion techniques and supply chain security enhancements.

### 4.1 Behavioral Mimicry

#### 4.1.1 Time-of-Day Constraints
- [ ] Implement configurable operating hours (e.g., 9 AM - 5 PM local time)
- [ ] Pause execution outside allowed hours
- [ ] **Tests:** Time-based execution tests

#### 4.1.2 Human-Like Connection Pooling
- [ ] Implement realistic Keep-Alive connection management
- [ ] Vary connection reuse patterns
- [ ] Add random think time between requests
- [ ] **Tests:** Connection pool behavior tests

### 4.2 VEX Integration
- [ ] Generate VEX documents alongside SBOM
- [ ] Mark unaffected dependencies to reduce false positives
- [ ] Automate VEX generation in GoReleaser
- [ ] **Deliverable:** VEX document per release

### 4.3 OpSec Improvements

#### 4.3.1 Self-Destruct Command (`trident burn`)
- [ ] Implement secure deletion of:
  - Configuration files
  - Logs
  - Cache
  - Binary itself (Linux/macOS only)
- [ ] Use secure overwrite (multiple passes)
- [ ] Log warning on Windows (binary deletion not supported)
- [ ] **Tests:** Mock filesystem tests, verify deletion

#### 4.3.2 DNS Leak Prevention Enhancements
- [ ] Verify SOCKS5 DNS routing
- [ ] Detect and warn about DNS leaks
- [ ] Provide leak test command
- [ ] **Tests:** DNS leak detection tests

### 4.4 Performance & Scalability
- [ ] Horizontal scaling support (distributed execution)
- [ ] Database backend for large result sets (SQLite/PostgreSQL)
- [ ] Streaming output for massive bulk processing
- [ ] **Tests:** Load tests with 100k+ inputs

### 4.5 Final Documentation & Polish
- [ ] Complete API documentation
- [ ] Create video tutorials (YouTube)
- [ ] Write blog posts for release announcements
- [ ] Publish to package managers (Homebrew, APT, Chocolatey)
- [ ] **Deliverable:** Comprehensive documentation site

**Phase 4 Deliverables:**
- ✅ Behavioral mimicry features
- ✅ VEX integration
- ✅ Self-destruct command
- ✅ Production-ready release (v1.1.0)
- ✅ Full documentation and marketing materials

---

## Phase 5: Community & Ecosystem (Ongoing)

**Goal:** Build community, gather feedback, and continuously improve.

### 5.1 Community Building
- [ ] Open source release on GitHub
- [ ] Create Discord/Slack community
- [ ] Accept and review community contributions
- [ ] Publish roadmap publicly

### 5.2 Plugin System (Exploratory)
- [ ] Design plugin architecture for custom services
- [ ] Implement plugin loader (Go plugins or external binaries)
- [ ] Create plugin development guide
- [ ] **Deliverable:** Plugin SDK

### 5.3 Cloud/Container Integration
- [ ] Official Docker image
- [ ] Kubernetes deployment manifests
- [ ] Cloud Functions integration (AWS Lambda, Google Cloud Functions)
- [ ] **Deliverable:** `docker pull trident/trident`

### 5.4 Continuous Improvement
- [ ] Monitor security advisories
- [ ] Keep dependencies updated (via Renovate)
- [ ] Performance profiling and optimization
- [ ] User feedback integration

---

## Success Metrics

### Phase 1 (MVP)
- ✅ 80%+ test coverage
- ✅ All CI/CD checks passing
- ✅ Zero critical security vulnerabilities (gosec, govulncheck)
- ✅ First GitHub release with SBOM + signed binaries
- ✅ 5 keyless services working end-to-end

### Phase 2
- ✅ 85%+ test coverage
- ✅ 4 additional services operational
- ✅ User guide published
- ✅ Performance benchmarks documented

### Phase 3
- ✅ 40+ services integrated
- ✅ 90%+ test coverage
- ✅ v1.0.0 release
- ✅ 100+ GitHub stars (community validation)

### Phase 4
- ✅ Production-ready release (v1.1.0)
- ✅ Listed on awesome-osint repositories
- ✅ Conference presentation (DEF CON, Black Hat, etc.)

---

## Risk Management

### Technical Risks
| Risk | Mitigation |
|------|-----------|
| API changes break services | Version API calls, implement graceful degradation, extensive tests |
| Rate limiting throttles users | Implement adaptive rate limiting, provide clear error messages |
| TLS fingerprinting detection | Stay updated with `utls` library, rotate profiles |
| Test coverage falls below 80% | Enforce coverage checks in CI, block PRs below threshold |

### Operational Risks
| Risk | Mitigation |
|------|-----------|
| API keys leaked in config | Enforce 0600 permissions, support env vars, secret masking |
| Dependency vulnerabilities | Renovate auto-updates, govulncheck in CI, quarterly audits |
| Community contributions lower quality | Strict PR review process, contributor guidelines, automated checks |

### Legal/Ethical Risks
| Risk | Mitigation |
|------|-----------|
| Tool misuse for illegal activity | Clear terms of use, educational focus, ethical guidelines |
| Violating API ToS | Respect rate limits, user-agent disclosure, PAP enforcement |
| OPSEC failures expose investigators | Extensive testing of proxy/defang/PAP features, documentation |

---

## Appendix: Technology Stack Summary

| Component | Technology | Justification |
|-----------|-----------|---------------|
| **Language** | Go 1.21+ | Performance, static compilation, cross-platform |
| **CLI Framework** | spf13/cobra | Industry standard, robust flag parsing |
| **Configuration** | spf13/viper | Multi-source config (file, env, flags) |
| **HTTP Client** | imroc/req v3 | Modern, feature-rich, testable |
| **Logging** | log/slog | Standard library, structured logging |
| **Testing** | testify + httpmock | Assertions + HTTP mocking |
| **Rate Limiting** | golang.org/x/time/rate | Token bucket, production-proven |
| **Output** | tablewriter | ASCII table formatting |
| **CI/CD** | GitHub Actions | Free, integrated with GitHub |
| **Linting** | golangci-lint | Comprehensive linter suite |
| **Security** | gosec + govulncheck | SAST + SCA scanning |
| **Releases** | GoReleaser | Automated multi-platform releases |
| **SBOM** | CycloneDX | Industry standard, supply chain security |
| **Signing** | Cosign | Sigstore, artifact signing |

---

## Next Steps

1. **Week 1-2:** Complete Phase 0 (Foundation & Infrastructure)
2. **Week 3:** Begin Phase 1.1-1.8 (Core framework)
3. **Week 4-5:** Implement Phase 1.9-1.12 (Services)
4. **Week 6:** Testing, documentation, and v0.1.0 release
5. **Quarterly Reviews:** Assess progress, adjust roadmap based on feedback

---

**Document Version:** 1.0  
**Last Updated:** 2026-01-09  
**Status:** Active Development
