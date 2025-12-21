# Trident - Quick PRD

**Objective:** Build Trident, a high-performance Go-based OSINT CLI tool that ports and evolves Python's Harpoon. Phase 1 MVP delivers 5 keyless services (DNS, ASN, Crt.sh, ThreatMiner, PGP) with production-grade OpSec features including PAP enforcement, output defanging, proxy support (HTTP/HTTPS/SOCKS5), and user-agent spoofing. The tool must support dual-input mode (CLI args + stdin), bulk processing with configurable concurrency (--concurrency flag), and three output formats (table/json/plain).

**Technical Stack & Architecture:** Use spf13/cobra for CLI, spf13/viper for config (~/.config/trident/config.yaml), imroc/req v3 for HTTP (no external SDKs), log/slog for logging, olekukonko/tablewriter for output. Implement strict dependency injection (constructor pattern, no globals), 80% minimum test coverage using stretchr/testify and jarcoal/httpmock. CI/CD via GitHub Actions with golangci-lint, gosec, govulncheck. Release automation with GoReleaser generating CycloneDX SBOMs and Cosign-signed binaries. Rate limiting via token bucket algorithm with jitter (golang.org/x/time/rate), respect HTTP 429/rate-limit headers. Cross-platform support (Linux/macOS/Windows) with strict input validation (domains/IPs/ASNs/hashes), HTTPS enforcement, and terminal injection prevention.

**Security & OpSec Requirements:** Enforce Permissible Actions Protocol (PAP) with command classification (RED/AMBER/GREEN/WHITE) and --pap-limit flag. Automatic output defanging (example[.]com) when PAP ≤ AMBER unless --no-defang is set. Support SOCKS5 with DNS leak prevention, rotating browser User-Agent strings, request jitter for WAF evasion. Implement strict secret management (0600 file permissions, env variable precedence, masked secrets in logs). Include self-cleanup "burn" command for forensics (note: binary self-deletion not supported on Windows Phase 1). All Phase 1 services are keyless (ASN via Team Cymru DNS, Crt.sh via JSON API, ThreatMiner REST API, PGP via HKP/HKPS keyservers).

---

*Generated with Clavix Planning Mode*
*Generated: 2025-12-21T10:40:28+01:00*
