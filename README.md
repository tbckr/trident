# Trident 🔱

[![CI](https://github.com/tbckr/trident/actions/workflows/ci.yml/badge.svg)](https://github.com/tbckr/trident/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tbckr/trident)](https://goreportcard.com/report/github.com/tbckr/trident)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Trident** is a high-performance, statically compiled CLI tool for Open Source Intelligence (OSINT) gathering, written in Go. It automates querying various threat intelligence, network, and identity platforms with a focus on speed, operational security (OpSec), and ease of use.

Trident is a port and evolution of the Python-based tool [Harpoon](https://github.com/Te-k/harpoon).

## 🚀 Key Features

- **Keyless Services**: Query DNS, ASN, Certificate Transparency (crt.sh), ThreatMiner, and PGP keys out-of-the-box without registration.
- **Operational Security (OpSec)**:
    - **PAP Enforcement**: Permissible Actions Protocol prevents accidental direct interaction with targets.
    - **Proxy Support**: Route all traffic through HTTP/HTTPS or SOCKS5 (with DNS leak protection).
    - **User-Agent Rotation**: Avoid fingerprinting with modern browser spoofing.
    - **Output Defanging**: Automatically defang indicators (e.g., `example[.]com`) to prevent accidental clicks.
- **Performance**: High-concurrency bulk processing via worker pools and efficient Go routines.
- **Flexible Output**: Supports human-readable tables, JSON for automation, and plain text for piping.
- **Self-Cleanup**: Securely "burn" configuration and artifacts with a single command.

## 📦 Installation

### From Binary

Download the latest binary for your operating system from the [Releases](https://github.com/tbckr/trident/releases) page.

### Using Go

```bash
go install github.com/tbckr/trident/cmd/trident@latest
```

## 🛠️ Quick Start

### Basic Commands

```bash
# DNS Lookup
trident dns example.com

# ASN Information (IP or ASN)
trident asn 8.8.8.8
trident asn AS15169

# Certificate Transparency Search
trident crtsh example.com

# ThreatMiner Intelligence (Domain, IP, or Hash)
trident threatminer example.com

# PGP Key Search
trident pgp user@example.com
```

### Bulk Processing

Trident excels at bulk processing via `stdin`:

```bash
cat domains.txt | trident dns --output json > results.json
```

## ⚙️ Configuration

Trident looks for a configuration file at `~/.config/trident/config.yaml`.

```yaml
proxy: socks5://127.0.0.1:9050
user_agent: "Mozilla/5.0 ... (custom)"
pap_limit: amber
concurrency: 20
```

See [config.example.yaml](./config.example.yaml) for a full list of options.

## 🛡️ Operational Security

### PAP Levels (Permissible Actions Protocol)

| Level | Description | Command Examples |
| :--- | :--- | :--- |
| **RED** | Offline/Local analysis only | (Future local processing) |
| **AMBER** | Passive queries via 3rd party APIs | `asn`, `crtsh`, `threatminer`, `pgp` |
| **GREEN** | Active queries (direct target interaction) | `dns` |
| **WHITE** | Unrestricted actions | N/A |

Enforce your safety limit using `--pap-limit`:
```bash
trident dns example.com --pap-limit amber # This will fail because DNS is GREEN
```

### Output Defanging

Trident automatically defangs output when the PAP limit is set to `amber` or `red`. You can also control this manually:

- `--defang`: Explicitly enable defanging.
- `--no-defang`: Explicitly disable defanging (useful for piping to other tools).

## 🏁 Global Flags

- `--config`: Path to config file (default: `~/.config/trident/config.yaml`).
- `--verbose, -v`: Enable debug logging.
- `--output, -o`: Output format (`text`|`json`|`plain`).
- `--proxy`: URL of a proxy (HTTP, HTTPS, SOCKS5).
- `--user-agent`: Override rotating User-Agent strings.
- `--pap-limit`: Set PAP enforcement level (`red`|`amber`|`green`|`white`).
- `--defang`: Enable output defanging.
- `--no-defang`: Disable output defanging.
- `--concurrency, -c`: Number of concurrent workers (default: 10).

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guidelines](CONTRIBUTING.md) (coming soon).

## 📄 License

Trident is licensed under the [MIT License](./LICENSE).
