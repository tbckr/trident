# **Product Requirements Document (PRD): Trident**

## **1\. Introduction**

**Trident** is a port and evolution of the Python-based OSINT tool [Harpoon](https://github.com/Te-k/harpoon), rewritten in **Go**. The goal is to create a high-performance, statically compiled CLI tool for Open Source Intelligence gathering. It automates querying various threat intelligence, network, and identity/social media platforms.

## **2\. Goals & Scope**

* **Language:** Go (Golang) for performance and ease of distribution (single binary).  
* **Architecture:** Modular design where each service (plugin) implements a common interface.  
* **Minimal Dependencies:** Avoid importing third-party SDKs for supported services. All API interactions should be implemented natively to keep the binary small and maintainable.  
* **Configuration:** Centralized configuration file (\~/.config/trident/config.yaml) managed via **Viper**.  
* **Phase 1 Focus:** Implementation of the core CLI framework and integration of the top 5 "Keyless" services to ensure immediate utility out-of-the-box.  
* **Non-Goals (Phase 1):**  
  * **VDR/VEX:** Automatic generation of Vulnerability Disclosure Reports (VDR) or VEX (Vulnerability Exploitability eXchange) documents. While basic SBOM generation is required, advanced vulnerability dispositioning is deferred to later phases.  
  * **GUI:** No graphical user interface; purely CLI-based.

## **3\. User Stories (Phase 1\)**

* **US-1:** As an analyst, I want to resolve domain names and retrieve ASN information without registering for API keys, so I can start an investigation immediately.  
* **US-2:** As an investigator, I want to find subdomains for a target domain using certificate transparency logs (crt.sh).  
* **US-3:** As a researcher, I want to check if an IP address or domain is associated with known threats using open threat intelligence data.  
* **US-4:** As an analyst, I want to search for PGP keys associated with an email address to identify potential targets.  
* **US-5:** As a user, I want a unified CLI command structure that is consistent across different services.  
* **US-6:** As an investigator, I want to mask my source IP address and tool fingerprint to avoid detection by the target infrastructure.  
* **US-7:** As an investigator, I want to set a strict operational limit (PAP) to ensure I do not accidentally interact directly with a target during a covert investigation.  
* **US-8:** As an analyst, I want the tool to automatically "defang" output (e.g., example\[.\]com) when operating under strict OpSec rules to prevent accidental clicks in the terminal.  
* **US-9:** As a power user, I want to pipe a list of domains or IPs into the tool via **stdin** to perform bulk analysis without running the command multiple times.  
* **US-10:** As a user, I want to control the level of parallelism (concurrency) to optimize performance or reduce network load.  
* **US-11:** As a power user, I want to explicitly disable output defanging via a flag to enable piping raw results to other tools, even when operating under strict OpSec limits.  
* **US-13:** As a power user, I want a "plain" output mode to retrieve results as a simple list (e.g., one domain per line), making it easy to pipe Trident's output into other CLI tools.

### **3.1 Future User Stories (Phase 3+)**

* **US-12 (Future):** As an investigator working on remote systems (headless/SSH), I want to encrypt all investigation artifacts (logs, results, cache) at rest, so that they are readable only with a session passphrase, protecting against server seizure or compromise.

## **4\. Technical Requirements & Stack**

### **4.1. Core Libraries**

* **CLI Framework:** [spf13/cobra](https://github.com/spf13/cobra)  
  * Used for command structure, flag parsing, and help generation.  
* **Configuration:** [spf13/viper](https://github.com/spf13/viper)  
  * Used for reading configuration from YAML files and environment variables.  
* **HTTP Client:** [imroc/req](https://github.com/imroc/req) (v3)  
  * Used for all HTTP interactions.  
  * **Constraint:** Do not use external API client libraries (SDKs). Implement raw HTTP requests using req to maintain control and minimize bloat.  
* **Logging:** [log/slog](https://pkg.go.dev/log/slog) (Standard Library)  
  * Used for structured logging throughout the application.  
  * No external logging libraries (like zap or logrus) permitted.  
* **Table Writer:** [olekukonko/tablewriter](https://github.com/olekukonko/tablewriter)  
  * Used for formatting CLI output into ASCII tables.  
* **Testing:**  
  * [stretchr/testify](https://github.com/stretchr/testify): Used for assertions, mocking, and test suites.  
  * [jarcoal/httpmock](https://github.com/jarcoal/httpmock): Used to mock HTTP responses at the transport layer, ensuring tests do not hit real external APIs.

### **4.2. Global Flags**

* \--config: Path to config file (default: \~/.config/trident/config.yaml).  
* \--verbose (-v): Sets slog level to Debug. Default is Info or Error.  
* \--output (-o): Output format. Supported values: text (default table view), json, plain (raw list). Example: \-o plain.  
* \--proxy: URL of a proxy to use (supports HTTP, HTTPS, SOCKS5). Example: socks5://127.0.0.1:9050.  
* \--user-agent: Override the User-Agent string.  
* \--pap-limit: Enforce a Permissible Actions Protocol limit. Commands exceeding this level will fail. Values: red (strict passive/offline), amber (passive/3rd party), green (active/direct), white (unrestricted). Default: white.  
* \--defang: Explicitly enable output defanging (independent of PAP).  
* \--no-defang: Explicitly disable output defanging. Overrides automatic defanging triggered by PAP limits (use with caution).  
* \--concurrency (-c): Number of concurrent workers for bulk processing and parallel lookups. Default: 10\.

### **4.3. Input Handling (Stdin & Args)**

* **Dual Input Mode:** All relevant commands (dns, asn, threatminer, etc.) must accept input from either CLI arguments OR standard input (stdin).  
* **Priority Logic:**  
  1. If CLI arguments are provided (e.g., trident dns example.com), process only these arguments.  
  2. If NO CLI arguments are provided, check if data is available on stdin (pipe check).  
  3. If stdin has data, process input line-by-line.  
* **Bulk Format:** Input from stdin is expected to be one entry per line. Empty lines and whitespace should be trimmed/ignored.  
* **Example:** cat domains.txt | trident dns

### **4.4. Software Architecture & Testability**

* **Dependency Injection (DI):**  
  * Use **constructor injection** for all services and components. Do NOT use global state or global singletons for core logic.  
  * Services must accept dependencies (like Logger, Config, HTTP Client) via interfaces defined in the domain layer.  
  * Example: NewThreatMinerService(client HttpClientInterface, logger \*slog.Logger) \*ThreatMinerService  
* **Interfaces:**  
  * Define Go interfaces for external interactions to allow easy mocking during tests.  
  * **HTTP Abstraction:** While imroc/req is the concrete client, usage should be wrapped or interfaced so tests can inject a mock client or transport (using httpmock).  
* **Test Coverage:**  
  * **Minimum Coverage:** A minimum code coverage of **80%** is mandatory for the entire application (not just core logic). This includes unit tests and integration tests with mocks.  
  * Integration tests for services should use recorded responses (fixtures) to prevent flaky tests due to network issues.

### **4.5. Development, CI/CD & Supply Chain Security**

* **Versioning:** Follow **Semantic Versioning (SemVer)** (e.g., v1.0.0, v1.1.0).  
* **Linting:** Use **golangci-lint** with strict settings. The build must fail on any linting error.  
* **Security Scanning (SAST/SCA):**  
  * **gosec:** Must run in the CI pipeline to detect security issues in the code (e.g., weak crypto, unhandled errors).  
  * **govulncheck:** Must run in the CI pipeline to detect known vulnerabilities in dependencies (Software Composition Analysis).  
* **CI Provider:** **GitHub Actions** must be used for all workflows (Testing, Linting, Building, Scanning).  
* **Release Automation:** Use **GoReleaser** to build, package, and publish releases (binaries/archives) to GitHub.  
* **Supply Chain Security:**  
  * **SBOM:** GoReleaser must be configured to generate a **Software Bill of Materials (SBOM)** using **CycloneDX** for every release artifact.  
  * **Signing:** Release binaries and checksums must be signed using **Cosign** to ensure integrity and authenticity.  
  * **Reproducible Builds:** Build configuration must ensure deterministic builds (e.g., using \-trimpath flags).  
* **Dependency Management:** Use **Renovate** to automatically monitor and create Pull Requests for dependency updates.

## **5\. Functional Requirements (Phase 1\)**

### **5.1. Selected Services (No API Key Required)**

These 5 services constitute the Minimum Viable Product (MVP) for Phase 1\.

#### **5.1.1. DNS (dns)**

* **Function:** Maps DNS information for a domain or IP.  
* **Input:** Domain name or IP address.  
* **Output:** A records, AAAA records, MX records, NS records, TXT records.  
* **Implementation:** Use Go's native net package (no HTTP required).  
* **PAP Level:** **GREEN** (Standard DNS queries are generally considered active if recursion exposes the request to the target's Nameserver, although caching resolvers mitigate this. For strict OpSec, this is Green).

#### **5.1.2. ASN (asn)**

* **Function:** Gather information on an Autonomous System Number (ASN) or IP owner.  
* **Input:** ASN string (e.g., "AS15169") or IP address.  
* **Output:** AS Description, Country, Registry, allocated prefixes (if available).  
* **Implementation:** Query Team Cymru via DNS TXT records using Go's native net.Resolver. **Do not** use os/exec to call dig. (Note: The Cymru DNS interface format is \<reversed-ip\>.origin.asn.cymru.com).  
* **PAP Level:** **AMBER** (Queries Cymru's infrastructure, not the target).

#### **5.1.3. Crt.sh (crtsh)**

* **Function:** Search Certificate Transparency logs via crt.sh.  
* **Input:** Domain name (e.g., example.com).  
* **Output:** List of subdomains, certificate issuance dates, CAs.  
* **Implementation:** HTTP GET to https://crt.sh/?q=%.\<domain\>\&output=json using imroc/req.  
* **PAP Level:** **AMBER** (Queries crt.sh infrastructure, not the target).

#### **5.1.4. ThreatMiner (threatminer)**

* **Function:** Contextual threat intelligence.  
* **Input:** Domain, IP, or Hash.  
* **Output:** Passive DNS history, associated malware hashes, Whois info.  
* **Implementation:** HTTP GET to https://api.threatminer.org/v2/ endpoints using imroc/req.  
* **PAP Level:** **AMBER** (Queries ThreatMiner infrastructure, not the target).

#### **5.1.5. PGP (pgp)**

* **Function:** Search for PGP keys.  
* **Input:** Email address or Name.  
* **Output:** Key ID, Fingerprint, User IDs, Creation Date.  
* **Implementation:** HTTP GET to HKP servers (e.g., https://keys.openpgp.org) using imroc/req.  
* **PAP Level:** **AMBER** (Queries Keyserver infrastructure, not the target).

## **6\. Non-Functional Requirements**

### **6.1. Performance & Concurrency**

* **Goroutines & Worker Pools:** Utilize Go routines to parallelize tasks. This is critical when processing bulk input from **stdin**. Implement a worker pool pattern where the pool size is determined by the \--concurrency flag.  
* **Concurrency Control:** Implement semaphores or bounded channels based on the configured concurrency limit to prevent local resource exhaustion and accidental DoS attacks against target infrastructure.

### **6.2. API Etiquette & Rate Limiting**

* **DoS Prevention:** Ensure the tool does not flood services. Implement a client-side rate limiter using the **Token Bucket** algorithm (recommended library: golang.org/x/time/rate) per service to respect polite usage policies.  
* **Standard Headers:** Automatically detect and respect standard HTTP Rate-Limit headers (X-RateLimit-Remaining, Retry-After, X-RateLimit-Reset) and HTTP 429 status codes. Pause execution or back off exponentially when limits are reached.

### **6.3. Reliability**

* **Error Handling:** Graceful failure if a service is unreachable. Use req's error handling features to retry transient network errors (with backoff) before failing.

### **6.4. Output**

* **Formats:** Support for both human-readable tables (using [olekukonko/tablewriter](https://github.com/olekukonko/tablewriter)), JSON output, and **plain text** output (raw list for piping).

### **6.5. Compatibility**

* **Operating Systems:** The application must compile and run natively on **Linux**, **macOS**, and **Windows**.  
  * Path handling must use filepath.Join to be OS-agnostic.  
  * Configuration location must respect OS standards (e.g., XDG on Linux, AppData on Windows) or fallback gracefully.

## **7\. Security Requirements**

### **7.1. Input Validation**

* **Strict Validation:** All CLI arguments must be validated against strict regex/syntax rules before processing or sending to external APIs.  
  * **Domains:** Must conform to standard hostname RFCs (e.g., no shell characters, correct TLD format).  
  * **IPs:** Must be valid IPv4 or IPv6 addresses (net.ParseIP).  
  * **ASNs:** Must match ^AS\\d+$ format.  
  * **Hashes:** Validate length and character set (MD5=32 hex, SHA1=40 hex, SHA256=64 hex).  
* **Sanitization:** Ensure no command injection or path traversal payloads are possible via input arguments.

### **7.2. Secret Management**

* **No Plaintext Secrets:** API keys and sensitive configuration values must **never** be printed to stdout, stderr, or log files.  
* **Masking:** If configuration dump/debug is required (e.g., via \--verbose), all secrets must be masked (e.g., \*\*\*\*\*\*\*\*).

### **7.3. Configuration Security**

* **File Permissions:** When creating the default configuration file, ensure strict file permissions (e.g., 0600 on Linux/macOS) so only the owner can read/write.  
* **Env Variable Precedence:** Viper should prefer environment variables (e.g., TRIDENT\_SHODAN\_KEY) over config file values to allow secret injection in CI/CD without persisting files.

### **7.4. Transport Security**

* **Enforced HTTPS:** All HTTP requests to external APIs must use HTTPS. Downgrade to HTTP is forbidden unless the target specifically does not support it (e.g., some legacy HKP servers), but modern defaults (HKPS) must be prioritized.  
* **TLS Verification:** Certificate verification must be enabled by default. Do not use InsecureSkipVerify unless explicitly overridden by a dangerous user flag (e.g., \--insecure).

### **7.5. Output Sanitization**

* **Terminal Injection:** Sanitize data received from external sources (e.g., WHOIS records, DNS TXT records) before printing to the terminal to prevent ANSI escape sequence injection attacks.

## **8\. Operational Security (OpSec) Requirements**

### **8.1. Network Anonymization**

* **Proxy Support:** The application must support routing **all** HTTP/HTTPS traffic through a proxy defined by a flag (--proxy) or configuration.  
* **Protocols:** Must support HTTP, HTTPS, and SOCKS5 (essential for Tor integration).  
* **DNS Leak Prevention:** When using a SOCKS5 proxy, DNS resolution must be performed remotely through the proxy (if the protocol supports it) to prevent local DNS leaks.

### **8.2. Browser Fingerprinting Protection**

* **User-Agent Spoofing:** By default, the application must **not** identify itself as "Trident" or "Go-http-client". It should use a generic, rotating list of modern browser User-Agent strings (e.g., Chrome on Windows, Firefox on Linux).  
* **Custom User-Agent:** Allow the user to override this with a specific string via flag (--user-agent) if needed for specific research authorization.

### **8.3. Traffic Analysis Protection (Jitter)**

* **Request Jitter:** To avoid detection by pattern-matching algorithms on firewalls (WAFs), the rate limiter (Token Bucket) should include a random "jitter" factor.  
  * *Implementation:* Instead of firing exactly every n milliseconds, introduce a random variation (e.g., ±20%) to the wait time between requests.

### **8.4. Permissible Actions Protocol (PAP) Integration**

* **Command Classification:** Every service/command must have a predefined PAP rating in the code:  
  * **RED:** Non-detectable (Offline/Local).  
  * **AMBER:** Detectable but not directly attributable to target (3rd Party APIs).  
  * **GREEN:** Active actions allowed (Direct interaction with target, e.g., direct DNS, Portscan, HTTP crawling).  
  * **WHITE:** Unrestricted.  
* **Enforcement:** The application must refuse to execute a command if its PAP rating is "higher" (more active) than the user's configured \--pap-limit.  
* **Visual Indicator:** The CLI help and execution logs should display the PAP level of the requested service.

### **8.5. Output Safety (Defanging)**

* **Requirement:** Prevent accidental execution or clicking of Indicators of Compromise (IOCs) in the terminal.  
* **Trigger:**  
  1. Enabled by default if \--pap-limit is **AMBER** or **RED**.  
  2. Enabled if \--defang is set.  
  3. **DISABLED** if \--no-defang is set (Overrides Trigger 1).  
* **Mechanism:**  
  * **Domains:** example.com \-\> example\[.\]com  
  * **IPs:** 1.2.3.4 \-\> 1.2.3\[.\]4  
  * **URLs:** https://malware.com \-\> hxxps://malware\[.\]com  
* **Format Interactions:**  
  * **JSON:** Remains raw by default (un-defanged) unless \--defang is explicitly set.  
  * **Plain:** Follows the same rules as text output. *Note: Users should use \--no-defang when piping plain output to other automated tools to avoid breaking parsers.*

### **8.6. Local Forensics & Cleanup**

* **Self-Cleanup Command:** Implement a burn or wipe subcommand (e.g., trident burn) that securely deletes all application artifacts, including configuration files, logs, and local caches.  
* **Binary Self-Deletion:** The command must attempt to remove the application binary itself from the file system.  
  * **Note:** Binary self-deletion is **not supported on Windows** in Phase 1 due to file locking mechanisms. The command should log a warning on Windows instead of failing.

## **10\. Future Outlook & Roadmap**

* **Phase 2:** Integration of caching mechanisms, offline databases (e.g., GeoIP), and additional passive services (Robtex, Quad9).  
* **Phase 3:**  
  * Integration of API-key dependent services (Shodan, Censys, VirusTotal).  
  * **Advanced OpSec \- Network:** Implementation of **TLS Fingerprinting Evasion** (JA3/JA4) using libraries like utls to mimic genuine browser handshakes.  
  * **Advanced OpSec \- Defense:** Integration of **Honeypot Detection** capabilities (passive canary checks) before executing active scans.  
  * **Advanced OpSec \- Host:** Robust, cross-platform implementation of **Core Dump Prevention** to ensure sensitive memory is never written to disk, even on complex OS configurations.  
  * **Encrypted Workspace & Artifacts (formerly US-12):** Implementation of a full session encryption mode for remote investigations.  
    * *Mechanism:* The user supplies a passphrase on startup.  
    * *Effect:* All logs, caches, and output files are written to an encrypted container/stream (AES-GCM/ChaCha20-Poly1305).  
    * *Goal:* Protect investigation data at rest on potentially compromised or seized remote servers.  
* **Phase 4 / Exploration:**  
  * **Behavioral Mimicry:** Implementation of "Time-of-Day" constraints and human-like connection pooling (Keep-Alive management).  
* **Advanced Supply Chain Security:** Integration of **VEX (Vulnerability Exploitability eXchange)** to reduce false positives in vulnerability scanners by explicitly flagging unaffected dependencies. This builds upon the SBOM foundation established in Phase 1\.

## **11\. Appendix: Service Categorization**

The following table categorizes the planned services based on their authentication and OpSec requirements.

| Service | Description | API Key Required? | PAP Level | Phase |
| :---- | :---- | :---- | :---- | :---- |
| **asn** | Gather information on an ASN | **No** | **AMBER** | 1 |
| **crtsh** | Search Certificate Transparency database | **No** | **AMBER** | 1 |
| **dns** | Map DNS information | **No** | **GREEN** | 1 |
| **pgp** | Search PGP key servers | **No** | **AMBER** | 1 |
| **threatminer** | Request ThreatMiner database | **No** | **AMBER** | 1 |
| **cache** | Request webpage cache (Archive.org etc.) | **No** (mostly) | **AMBER** | 2 |
| **quad9** | Check if blocked by Quad9 | **No** (DNS) | **AMBER** | 2 |
| **tor** | Check Tor exit node list | **No** | **AMBER** | 2 |
| **robtex** | Search Robtex | **No** (Limited/Scrape) | **AMBER** | 2 |
| **umbrella** | Umbrella Top 1 Million check | **No** (CSV List) | **RED** (if local) | 2 |
| **binaryedge** | Request BinaryEdge API | Yes | **AMBER** | 3 |
| **censys** | Request Censys database | Yes | **AMBER** | 3 |
| **certspotter** | Get certificates from SSLMate | Yes | **AMBER** | 3 |
| **circl** | CIRCL passive DNS | Yes (Auth required) | **AMBER** | 3 |
| **fullcontact** | Full Contact API | Yes | **AMBER** | 3 |
| **github** | Github API | Yes (for rate limits) | **AMBER** | 3 |
| **greynoise** | GreyNoise API | Yes | **AMBER** | 3 |
| **hibp** | Have I Been Pwned API | Yes | **AMBER** | 3 |
| **hunter** | Hunter.io API | Yes | **AMBER** | 3 |
| **hybrid** | Hybrid Analysis platform | Yes | **AMBER** | 3 |
| **ipinfo** | ipinfo.io information | Yes | **AMBER** | 3 |
| **ip2locationio** | IP2Location.io information | Yes | **AMBER** | 3 |
| **koodous** | Koodous API | Yes | **AMBER** | 3 |
| **malshare** | MalShare database | Yes | **AMBER** | 3 |
| **misp** | MISP server API | Yes | **AMBER** | 3 |
| **numverify** | NumVerify API | Yes | **AMBER** | 3 |
| **opencage** | OpenCage Geocoding | Yes | **AMBER** | 3 |
| **otx** | AlienVault OTX | Yes | **AMBER** | 3 |
| **permacc** | Perma.cc API | Yes | **GREEN** (Scans URL) | 3 |
| **pt** | Passive Total database | Yes | **AMBER** | 3 |
| **pulsedive** | PulseDive API | Yes | **AMBER** | 3 |
| **safebrowsing** | Google Safe Browsing | Yes | **AMBER** | 3 |
| **securitytrails** | SecurityTrails database | Yes | **AMBER** | 3 |
| **shodan** | Shodan API | Yes | **AMBER** | 3 |
| **spyonweb** | SpyOnWeb API | Yes | **AMBER** | 3 |
| **telegram** | Telegram API | Yes | **AMBER** | 3 |
| **threatcrowd** | ThreatCrowd API | Yes | **AMBER** | 3 |
| **threatgrid** | Threat Grid API | Yes | **AMBER** | 3 |
| **totalhash** | Total Hash API | Yes | **AMBER** | 3 |
| **twitter** | Twitter API | Yes | **AMBER** | 3 |
| **urlhaus** | urlhaus.abuse.ch API | Yes (Auth-Key) | **AMBER** | 3 |
| **urlscan** | urlscan.io | Yes | **GREEN** (Scans URL) | 3 |
| **vt** | Virus Total API | Yes | **AMBER** | 3 |
| **xforce** | IBM Xforce Exchange API | Yes | **AMBER** | 3 |
| **zetalytics** | Zetalytics database | Yes | **AMBER** | 3 |

