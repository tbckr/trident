# Trident Implementation Plan

**Version:** 1.0  
**Date:** 2025-12-20  
**Phase:** Phase 1 (MVP)

---

## Overview

This implementation plan breaks down the development of Trident (Phase 1) into manageable work packages and individual tasks. Each work package represents a coherent unit of functionality that can be developed, tested, and integrated independently.

---

## Work Package 1: Project Foundation & Scaffolding

**Goal:** Establish the basic project structure, dependencies, and development tooling.

**Priority:** P0 (Blocker for all other work)

### Tasks

- [x] **WP1-T1:** Initialize Go module and project structure
  - Create `cmd/trident/main.go` entry point
  - Set up basic directory structure: `internal/`, `pkg/`, `docs/`, `testdata/`
  - Initialize `go.mod` with module name

- [x] **WP1-T2:** Add core dependencies
  - Add `spf13/cobra` for CLI framework
  - Add `spf13/viper` for configuration
  - Add `imroc/req` (v3) for HTTP client
  - Add `olekukonko/tablewriter` for output formatting
  - Add `golang.org/x/time/rate` for rate limiting

- [x] **WP1-T3:** Add testing dependencies
  - Add `stretchr/testify` for assertions
  - Add `jarcoal/httpmock` for HTTP mocking

- [x] **WP1-T4:** Configure development tooling
  - Create `.golangci.yml` with strict linting rules
  - Add `Makefile` with targets: build, test, lint, clean
  - Create `.editorconfig` for consistent formatting

---

## Work Package 2: Core CLI Framework

**Goal:** Implement the foundational CLI structure following Go best practices (no global state, dependency injection).

**Priority:** P0 (Blocker for all commands)

**Dependencies:** WP1

### Tasks

- [x] **WP2-T1:** Implement `main.go` with the run function pattern
  - Create minimal `main()` that initializes context, logger, and calls `run()`
  - Implement dynamic logging with `slog.LevelVar`
  - Handle exit codes and signal handling in `run()`

- [x] **WP2-T2:** Create root command constructor
  - Implement `NewRootCmd(logger, levelVar, ...)` following DI principles
  - Add global flags: `--config`, `--verbose`, `--output`, `--proxy`, `--user-agent`, `--pap-limit`, `--defang`, `--no-defang`, `--concurrency`
  - Bind flags to options struct (GoReleaser-style pattern)

- [x] **WP2-T3:** Implement PersistentPreRunE for root command
  - Check `--verbose` flag and set log level dynamically via `levelVar.Set(slog.LevelDebug)`
  - Validate global flag combinations (e.g., `--defang` vs `--no-defang`)

- [x] **WP2-T4:** Wire up cobra command execution
  - Call `rootCmd.SetArgs()`, `SetIn()`, `SetOut()`, `SetErr()` in `run()`
  - Execute root command with context: `rootCmd.ExecuteContext(ctx)`

---

## Work Package 3: Configuration Management

**Goal:** Implement centralized configuration using Viper with proper security controls.

**Priority:** P0

**Dependencies:** WP1, WP2

### Tasks

- [x] **WP3-T1:** Define configuration schema
  - Create `internal/config/config.go` with Config struct
  - Define fields for API keys, default settings, PAP limits, etc.

- [x] **WP3-T2:** Implement Viper initialization
  - Configure default config path: `~/.config/trident/config.yaml`
  - Respect OS-specific paths (XDG on Linux, AppData on Windows)
  - Use `filepath.Join` for cross-platform compatibility

- [ ] **WP3-T3:** Implement environment variable precedence
  - Configure Viper to read env vars with prefix `TRIDENT_`
  - Ensure env vars override config file values

- [ ] **WP3-T4:** Implement config file creation with secure permissions
  - Create default config file if missing
  - Set file permissions to 0600 on Linux/macOS
  - Generate sample config with masked secrets

- [ ] **WP3-T5:** Implement secret masking
  - Mask secrets when printing config in `--verbose` mode
  - Replace secret values with `********` in debug output

---

## Work Package 4: Input Handling (Stdin & Args)

**Goal:** Implement dual input mode for all commands (CLI args or stdin pipe).

**Priority:** P1

**Dependencies:** WP2

### Tasks

- [ ] **WP4-T1:** Create input abstraction
  - Define `InputReader` interface or helper function
  - Implement priority logic: CLI args → stdin check → process stdin line-by-line

- [ ] **WP4-T2:** Implement stdin detection
  - Use `os.Stdin.Stat()` to check if data is piped
  - Handle empty lines and whitespace trimming

- [ ] **WP4-T3:** Implement line-by-line processing
  - Use `bufio.Scanner` to read stdin
  - Trim whitespace and skip empty lines
  - Return slice of inputs for processing

- [ ] **WP4-T4:** Add unit tests for input handling
  - Test CLI args only
  - Test stdin only
  - Test priority (args override stdin)

---

## Work Package 5: Output Formatting & Defanging

**Goal:** Implement multiple output formats (text/table, json, plain) with OpSec-aware defanging.

**Priority:** P1

**Dependencies:** WP2

### Tasks

- [ ] **WP5-T1:** Create output formatter abstraction
  - Define `Formatter` interface with methods: `Format(data interface{}) error`
  - Implement formatters: TextFormatter (tablewriter), JSONFormatter, PlainFormatter

- [ ] **WP5-T2:** Implement defanging logic
  - Create defang function: `defang(input string) string`
  - Handle domains: `example.com` → `example[.]com`
  - Handle IPs: `1.2.3.4` → `1.2.3[.]4`
  - Handle URLs: `https://malware.com` → `hxxps://malware[.]com`

- [ ] **WP5-T3:** Implement defang trigger logic
  - Enable by default if `--pap-limit` is AMBER or RED
  - Enable if `--defang` is set
  - Disable if `--no-defang` is set (override)

- [ ] **WP5-T4:** Integrate defanging with formatters
  - Apply defanging to text/plain output based on triggers
  - JSON remains raw by default unless `--defang` explicitly set

- [ ] **WP5-T5:** Implement output sanitization (ANSI escape prevention)
  - Sanitize external data before printing to prevent terminal injection

---

## Work Package 6: HTTP Client & Network Layer

**Goal:** Set up HTTP client with proxy support, rate limiting, and OpSec features.

**Priority:** P1

**Dependencies:** WP1, WP3

### Tasks

- [ ] **WP6-T1:** Create HTTP client wrapper
  - Abstract `imroc/req` behind an interface for testability
  - Define `HttpClientInterface` with methods: `Get()`, `Post()`, etc.

- [ ] **WP6-T2:** Implement proxy support
  - Configure `req` client to use `--proxy` flag (HTTP, HTTPS, SOCKS5)
  - Ensure DNS resolution goes through SOCKS5 proxy to prevent leaks

- [ ] **WP6-T3:** Implement User-Agent spoofing
  - Create list of modern browser User-Agent strings
  - Rotate or randomize selection
  - Allow override via `--user-agent` flag

- [ ] **WP6-T4:** Implement rate limiting (Token Bucket)
  - Use `golang.org/x/time/rate` for rate limiting per service
  - Add random jitter (±20%) to prevent pattern detection

- [ ] **WP6-T5:** Implement HTTP rate limit header detection
  - Detect `X-RateLimit-Remaining`, `Retry-After`, `X-RateLimit-Reset`
  - Auto-pause or exponential backoff on HTTP 429

- [ ] **WP6-T6:** Enforce HTTPS-only by default
  - Validate all API endpoints use HTTPS
  - Prevent downgrade unless explicitly allowed

---

## Work Package 7: OpSec Features (PAP, Concurrency, Burn)

**Goal:** Implement Permissible Actions Protocol (PAP), concurrency control, and self-cleanup.

**Priority:** P1

**Dependencies:** WP2, WP3

### Tasks

- [ ] **WP7-T1:** Define PAP levels and command classification
  - Create constants: RED, AMBER, GREEN, WHITE
  - Add PAP metadata to each service/command

- [ ] **WP7-T2:** Implement PAP enforcement
  - Check command PAP level against `--pap-limit` before execution
  - Refuse execution if command PAP > configured limit
  - Log/display PAP level in help and execution output

- [ ] **WP7-T3:** Implement concurrency control
  - Create worker pool pattern with semaphore or bounded channels
  - Pool size determined by `--concurrency` flag (default: 10)
  - Apply to bulk stdin processing

- [ ] **WP7-T4:** Implement `burn` command for self-cleanup
  - Delete config files (`~/.config/trident/`)
  - Delete logs and caches
  - Attempt binary self-deletion (Linux/macOS only)
  - Log warning on Windows (file locking prevents self-deletion)

---

## Work Package 8: Input Validation & Security

**Goal:** Implement strict input validation to prevent injection attacks.

**Priority:** P1

**Dependencies:** WP2

### Tasks

- [ ] **WP8-T1:** Create validation utilities
  - Implement `ValidateDomain(domain string) error`
  - Implement `ValidateIP(ip string) error` using `net.ParseIP`
  - Implement `ValidateASN(asn string) error` (regex: `^AS\d+$`)
  - Implement `ValidateHash(hash string, hashType string) error`

- [ ] **WP8-T2:** Apply validation to all user inputs
  - Validate CLI arguments before processing
  - Validate stdin inputs before processing
  - Return clear error messages on validation failure

- [ ] **WP8-T3:** Add unit tests for validation
  - Test valid inputs (happy path)
  - Test invalid inputs (malicious payloads, edge cases)

---

## Work Package 9: Service Implementation - DNS

**Goal:** Implement the DNS service using native Go `net` package.

**Priority:** P1

**Dependencies:** WP2, WP4, WP5, WP8

### Tasks

- [ ] **WP9-T1:** Create DNS service structure
  - Define `internal/services/dns/` package
  - Create `DNSService` struct with logger dependency

- [ ] **WP9-T2:** Implement DNS lookup logic
  - Use `net.Resolver` for A, AAAA, MX, NS, TXT lookups
  - Handle both domain and IP inputs (reverse DNS for IPs)

- [ ] **WP9-T3:** Create DNS command
  - Implement `NewDNSCmd(logger, ...)` constructor
  - Wire up input handling (args + stdin)
  - Wire up output formatting

- [ ] **WP9-T4:** Set PAP level to GREEN
  - Add PAP metadata to command

- [ ] **WP9-T5:** Write tests
  - Unit tests with mocked resolver (if possible)
  - Integration tests with real DNS (separate from unit tests)
  - Test multiple input formats

---

## Work Package 10: Service Implementation - ASN

**Goal:** Implement the ASN service using Team Cymru DNS queries.

**Priority:** P1

**Dependencies:** WP2, WP4, WP5, WP8

### Tasks

- [ ] **WP10-T1:** Create ASN service structure
  - Define `internal/services/asn/` package
  - Create `ASNService` struct with logger dependency

- [ ] **WP10-T2:** Implement Team Cymru DNS query logic
  - Use `net.Resolver` for TXT record lookups
  - Query format: `<reversed-ip>.origin.asn.cymru.com` for IPs
  - Query format for ASN: `AS<number>.asn.cymru.com`
  - Parse TXT response (format: `ASN | CIDR | Country | Registry | Allocated Date`)

- [ ] **WP10-T3:** Create ASN command
  - Implement `NewASNCmd(logger, ...)` constructor
  - Handle both IP and ASN inputs
  - Wire up formatting

- [ ] **WP10-T4:** Set PAP level to AMBER

- [ ] **WP10-T5:** Write tests
  - Mock DNS responses using `httpmock` or custom resolver
  - Test IP → ASN lookup
  - Test ASN → info lookup

---

## Work Package 11: Service Implementation - Crt.sh

**Goal:** Implement Certificate Transparency log search via crt.sh API.

**Priority:** P1

**Dependencies:** WP2, WP4, WP5, WP6, WP8

### Tasks

- [ ] **WP11-T1:** Create Crt.sh service structure
  - Define `internal/services/crtsh/` package
  - Create `CrtshService` struct with HTTP client and logger dependencies

- [ ] **WP11-T2:** Implement crt.sh API client
  - HTTP GET to `https://crt.sh/?q=%.{domain}&output=json`
  - Parse JSON response
  - Extract subdomains, issuance dates, CAs

- [ ] **WP11-T3:** Create crtsh command
  - Implement `NewCrtshCmd(logger, httpClient, ...)` constructor
  - Accept domain inputs
  - Wire up formatting

- [ ] **WP11-T4:** Set PAP level to AMBER

- [ ] **WP11-T5:** Write tests
  - Use `httpmock` to mock crt.sh responses
  - Test successful response parsing
  - Test error handling (network failures, API errors)

---

## Work Package 12: Service Implementation - ThreatMiner

**Goal:** Implement ThreatMiner threat intelligence service.

**Priority:** P1

**Dependencies:** WP2, WP4, WP5, WP6, WP8

### Tasks

- [ ] **WP12-T1:** Create ThreatMiner service structure
  - Define `internal/services/threatminer/` package
  - Create `ThreatMinerService` struct with HTTP client and logger

- [ ] **WP12-T2:** Implement ThreatMiner API client
  - HTTP GET to `https://api.threatminer.org/v2/` endpoints
  - Implement endpoints for: domain, IP, hash lookups
  - Parse JSON responses

- [ ] **WP12-T3:** Create threatminer command
  - Implement `NewThreatMinerCmd(logger, httpClient, ...)` constructor
  - Support domain, IP, hash inputs
  - Wire up formatting

- [ ] **WP12-T4:** Set PAP level to AMBER

- [ ] **WP12-T5:** Write tests
  - Mock API responses with `httpmock`
  - Test each endpoint (domain, IP, hash)
  - Test error handling

---

## Work Package 13: Service Implementation - PGP

**Goal:** Implement PGP keyserver search.

**Priority:** P1

**Dependencies:** WP2, WP4, WP5, WP6, WP8

### Tasks

- [ ] **WP13-T1:** Create PGP service structure
  - Define `internal/services/pgp/` package
  - Create `PGPService` struct with HTTP client and logger

- [ ] **WP13-T2:** Implement PGP keyserver client
  - HTTP GET to `https://keys.openpgp.org` (HKP/HKPS)
  - Search by email or name
  - Parse response (may be HTML or JSON depending on server)
  - Extract Key ID, Fingerprint, User IDs, Creation Date

- [ ] **WP13-T3:** Create pgp command
  - Implement `NewPGPCmd(logger, httpClient, ...)` constructor
  - Accept email or name inputs
  - Wire up formatting

- [ ] **WP13-T4:** Set PAP level to AMBER

- [ ] **WP13-T5:** Write tests
  - Mock keyserver responses
  - Test email search
  - Test name search
  - Test error handling

---

## Work Package 14: Testing Infrastructure & Coverage

**Goal:** Establish comprehensive test coverage (minimum 80%) and testing patterns.

**Priority:** P1

**Dependencies:** All service WPs (9-13)

### Tasks

- [ ] **WP14-T1:** Set up test infrastructure
  - Create `testdata/` directories for golden files
  - Document black-box testing pattern (separate `_test` packages)
  - Document table-driven test pattern

- [ ] **WP14-T2:** Implement integration test helpers
  - Create test fixtures for HTTP responses
  - Create helpers for mocking HTTP clients
  - Document golden file usage

- [ ] **WP14-T3:** Write integration tests for all services
  - Use recorded/mocked responses (avoid real API calls)
  - Test happy paths
  - Test error paths (network failures, malformed responses)

- [ ] **WP14-T4:** Measure and enforce 80% coverage
  - Add coverage reporting to Makefile
  - Run `go test -cover ./...`
  - Identify and fill coverage gaps

- [ ] **WP14-T5:** Add parallel test support
  - Ensure tests use `t.Parallel()` where safe
  - Verify no global state prevents parallel execution

---

## Work Package 15: CI/CD Pipeline (GitHub Actions)

**Goal:** Set up automated testing, linting, and security scanning in CI.

**Priority:** P2

**Dependencies:** WP1, WP14

### Tasks

- [ ] **WP15-T1:** Create test workflow
  - `.github/workflows/test.yml`
  - Run on: push, pull_request
  - Matrix: Linux, macOS, Windows
  - Matrix: Go versions (latest, latest-1)

- [ ] **WP15-T2:** Create lint workflow
  - `.github/workflows/lint.yml`
  - Run `golangci-lint`
  - Fail build on any linting error

- [ ] **WP15-T3:** Create security scanning workflow
  - `.github/workflows/security.yml`
  - Run `gosec` for SAST
  - Run `govulncheck` for SCA (dependency vulnerabilities)

- [ ] **WP15-T4:** Add code coverage reporting
  - Integrate coverage tool (e.g., codecov.io or coveralls)
  - Upload coverage reports from test workflow
  - Display coverage badge in README

---

## Work Package 16: Release Automation (GoReleaser)

**Goal:** Automate building, packaging, signing, and publishing releases.

**Priority:** P2

**Dependencies:** WP1, WP15

### Tasks

- [ ] **WP16-T1:** Create GoReleaser configuration
  - `.goreleaser.yml`
  - Configure build targets: Linux, macOS, Windows (amd64, arm64)
  - Use `-trimpath` for reproducible builds

- [ ] **WP16-T2:** Configure SBOM generation
  - Enable CycloneDX SBOM generation in GoReleaser
  - Include SBOM in release artifacts

- [ ] **WP16-T3:** Configure Cosign signing
  - Sign binaries and checksums with Cosign
  - Document keyless signing setup (GitHub OIDC)

- [ ] **WP16-T4:** Create release workflow
  - `.github/workflows/release.yml`
  - Trigger on: tag push (`v*`)
  - Run GoReleaser to build and publish to GitHub Releases

- [ ] **WP16-T5:** Test release process
  - Create test tag and verify artifacts
  - Verify SBOM and signatures are present

---

## Work Package 17: Dependency Management (Renovate)

**Goal:** Automate dependency updates with Renovate.

**Priority:** P3 (Nice-to-have for Phase 1)

**Dependencies:** WP1

### Tasks

- [ ] **WP17-T1:** Create Renovate configuration
  - `renovate.json`
  - Configure Go module updates
  - Configure GitHub Actions updates
  - Set auto-merge rules for minor/patch updates

- [ ] **WP17-T2:** Enable Renovate on repository
  - Install Renovate GitHub App
  - Verify PRs are created for dependencies

---

## Work Package 18: Documentation

**Goal:** Write comprehensive user and developer documentation.

**Priority:** P2

**Dependencies:** All feature WPs

### Tasks

- [ ] **WP18-T1:** Write README.md
  - Project overview
  - Installation instructions
  - Quick start guide
  - Feature list
  - Example commands

- [ ] **WP18-T2:** Write CONTRIBUTING.md
  - Development setup
  - Testing guidelines
  - PR process
  - Code style (reference AGENTS.md)

- [ ] **WP18-T3:** Write command documentation
  - Usage examples for each service
  - Global flag reference
  - OpSec best practices
  - PAP level guide

- [ ] **WP18-T4:** Create man pages (optional)
  - Generate man pages from Cobra (if time permits)

---

## Work Package 19: Final Integration & End-to-End Testing

**Goal:** Integrate all components and perform end-to-end validation.

**Priority:** P0 (Before release)

**Dependencies:** All WPs

### Tasks

- [ ] **WP19-T1:** Integration testing
  - Test all services end-to-end with real APIs (manual verification)
  - Verify stdin piping works for all commands
  - Test all output formats (text, json, plain)

- [ ] **WP19-T2:** OpSec feature validation
  - Test PAP enforcement (try to run GREEN command with AMBER limit)
  - Test defanging in all modes
  - Test proxy support (HTTP, SOCKS5)
  - Test User-Agent spoofing

- [ ] **WP19-T3:** Cross-platform testing
  - Build and run on Linux
  - Build and run on macOS
  - Build and run on Windows
  - Verify config paths are OS-appropriate

- [ ] **WP19-T4:** Performance testing
  - Test bulk processing with large stdin input
  - Verify concurrency flag works correctly
  - Verify rate limiting prevents API overload

- [ ] **WP19-T5:** Final cleanup
  - Remove dead code
  - Fix any remaining linting issues
  - Update documentation with final behaviors

---

## Task Prioritization

**P0 (Critical Path):** Must be completed for MVP  
**P1 (High Priority):** Core features for Phase 1  
**P2 (Medium Priority):** Important but not blocking  
**P3 (Low Priority):** Nice-to-have for Phase 1

---

## Estimated Timeline (Rough)

Assuming a single developer working part-time:

- **WP1-2:** Project setup & CLI framework (1-2 weeks)
- **WP3-8:** Core infrastructure (2-3 weeks)
- **WP9-13:** Service implementations (2-3 weeks)
- **WP14:** Testing (1-2 weeks)
- **WP15-16:** CI/CD & Release (1 week)
- **WP17-18:** Dependency management & Docs (1 week)
- **WP19:** Final integration (1 week)

**Total:** ~8-12 weeks for Phase 1 MVP

---

## Success Criteria

Phase 1 is considered complete when:

1. All 5 services (dns, asn, crtsh, threatminer, pgp) are implemented and tested
2. All global flags work correctly
3. Stdin piping works for all commands
4. OpSec features (PAP, defanging, proxy) are functional
5. Code coverage ≥ 80%
6. All CI/CD checks pass (lint, test, security)
7. Release automation produces signed binaries with SBOMs
8. Documentation is complete and accurate

---

## Notes for Implementation

1. **Follow AGENTS.md:** Strictly adhere to Go best practices outlined in the project's AGENTS.md (no global state, dependency injection, simplicity over cleverness).
2. **Think First:** Review existing code and research official docs before implementing each work package.
3. **Test as You Go:** Write tests alongside implementation, not after.
4. **Incremental Progress:** Each work package should result in working, tested code that can be merged.
5. **Refactor Ruthlessly:** If you encounter code smells, fix them immediately. Leave the codebase better than you found it.
