# Project Structure Templates

## CLI Application Structure

```
myapp/
├── cmd/
│   └── myapp/
│       └── main.go              # Ultra-simple main
├── internal/
│   ├── cli/
│   │   ├── root.go              # Root command constructor
│   │   ├── serve.go             # Subcommand constructors
│   │   └── version.go
│   ├── config/
│   │   └── config.go            # Configuration structs
│   └── service/
│       ├── service.go           # Business logic
│       └── service_test.go      # Black-box tests
├── go.mod
├── go.sum
├── .golangci.yml                # Linter config
├── .goreleaser.yml              # Release config
└── README.md
```

## HTTP Application Structure

```
myapp/
├── cmd/
│   └── myapp/
│       └── main.go              # Ultra-simple main
├── internal/
│   ├── server/
│   │   ├── server.go            # Server struct and constructor
│   │   ├── routes.go            # Route definitions
│   │   ├── handlers.go          # HTTP handlers
│   │   ├── middleware.go        # Middleware
│   │   └── server_test.go       # Black-box tests
│   ├── service/
│   │   ├── items.go             # Business logic
│   │   └── items_test.go        # Black-box tests
│   └── repository/
│       ├── repository.go        # Data access interface
│       └── postgres.go          # Implementation
├── templates/
│   ├── base.html                # Base template
│   ├── index.html
│   └── item.html
├── static/
│   ├── css/
│   └── js/
├── go.mod
├── go.sum
├── .golangci.yml
├── .goreleaser.yml
└── README.md
```

## Library Structure

```
mylib/
├── mylib.go                     # Main package file
├── mylib_test.go                # Black-box tests
├── options.go                   # Configuration options
├── internal/
│   └── implementation/          # Private implementation details
├── go.mod
├── go.sum
├── .golangci.yml
└── README.md
```

## Shared Patterns

### go.mod

```go
module github.com/user/myapp

go 1.23

require (
    github.com/prometheus/client_golang v1.19.0
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.2
    github.com/stretchr/testify v1.9.0
)
```

### .golangci.yml

```yaml
version: "2"
run:
  go: "1.25"
linters:
  enable:
    - bodyclose
    - copyloopvar
    - depguard
    - forbidigo
    - gochecknoglobals
    - goconst
    - gocritic
    - godoclint
    - goerr113
    - gosec
    - misspell
    - nilerr
    - noctx
    - nolintlint
    - perfsprint
    - revive
    - tagliatelle
    - testifylint
    - thelper
    - tparallel
    - unconvert
    - unparam
    - usetesting
    - wastedassign
  settings:
    depguard:
      rules:
        main:
          deny:
            - pkg: github.com/pkg/errors
              desc: use stdlib instead
            - pkg: math/rand$
              desc: use math/rand/v2 instead
    forbidigo:
      forbid:
        - pattern: ioutil\.*
    gocritic:
      disabled-checks:
        - appendAssign
    perfsprint:
      int-conversion: false
      err-error: false
      errorf: true
      sprintf1: false
      strconcat: false
    revive:
      enable-all-rules: false
      rules:
        - name: blank-imports
        - name: context-as-argument
        - name: context-keys-type
        - name: comment-spacings
        - name: dot-imports
        - name: empty-block
        - name: empty-lines
        - name: error-naming
        - name: error-return
        - name: error-strings
        - name: errorf
        - name: increment-decrement
        - name: indent-error-flow
        - name: modifies-value-receiver
        - name: range
        - name: receiver-naming
        - name: redefines-builtin-id
        - name: superfluous-else
        - name: time-naming
        - name: unexported-return
        - name: unreachable-code
        - name: unused-parameter
        - name: var-declaration
        - name: var-naming
          arguments:
            - []
            - []
            - - skip-package-name-collision-with-go-std: true
    staticcheck:
      checks:
        - all
        - -SA1019
    tagliatelle:
      case:
        rules:
          json: snake
          yaml: snake
        use-field-name: false
    testifylint:
      enable-all: true
      disable:
        - error-is-as
    usetesting:
      context-background: true
      context-todo: true
      os-chdir: true
      os-mkdir-temp: true
      os-setenv: true
      os-create-temp: true
      os-temp-dir: true
  exclusions:
    generated: lax
    presets:
      - common-false-positives
      - legacy
      - std-error-handling
    rules:
      - linters:
          - noctx
          - perfsprint
        path: _test\.go
    paths:
      - third_party$
      - builtin$
      - examples$
formatters:
  enable:
    - gofumpt
    - goimports
  exclusions:
    generated: lax
    paths:
      - third_party$
      - builtin$
      - examples$
```

**Key linters enabled:**
- `bodyclose` - checks HTTP response body is closed
- `copyloopvar` - detects loop variable capture issues
- `depguard` - blocks deprecated packages (pkg/errors, math/rand)
- `forbidigo` - forbids ioutil.* (use os/io instead)
- `gochecknoglobals` - enforces no global variables
- `goconst` - finds repeated strings that should be constants
- `gocritic` - comprehensive meta-linter
- `godoclint` - checks doc comment formatting
- `goerr113` - checks error wrapping with %w
- `gosec` - security-focused linter
- `testifylint` - testify best practices
- `usetesting` - enforces testing.T helpers

**Formatters:**
- `gofumpt` - stricter gofmt
- `goimports` - manages imports automatically


### .goreleaser.yml

```yaml
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2

env:
  - GO111MODULE=on

before:
  hooks:
    - go mod tidy

snapshot:
  version_template: "{{ incpatch .Version }}-rc"

gomod:
  proxy: true

report_sizes: true

metadata:
  mod_timestamp: "{{ .CommitTimestamp }}"

builds:
  - main: ./cmd/myapp
    binary: myapp
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - "386"
      - amd64
      - arm
      - arm64
    goarm:
      - "7"
    ignore:
      - goos: windows
        goarch: arm
    mod_timestamp: "{{ .CommitTimestamp }}"
    flags:
      - -trimpath
    ldflags:
      - -s -w

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  use: github
  format: "{{ .SHA }}: {{ .Message }}{{ with .AuthorUsername }} (@{{ . }}){{ end }}"
  filters:
    exclude:
      - "^test:"
      - "^test\\("
      - "^chore\\(deps\\): "
      - "^(build|ci): "
      - "merge conflict"
      - Merge pull request
      - Merge remote-tracking branch
      - Merge branch
      - go mod tidy
  groups:
    - title: "New Features"
      regexp: '^.*?feat(\(.+\))??!?:.+$'
      order: 100
    - title: "Security updates"
      regexp: '^.*?sec(\(.+\))??!?:.+$'
      order: 150
    - title: "Bug fixes"
      regexp: '^.*?(fix|refactor)(\(.+\))??!?:.+$'
      order: 200
    - title: "Documentation updates"
      regexp: ^.*?docs?(\(.+\))??!?:.+$
      order: 400
    - title: Other work
      order: 9999

archives:
  - name_template: >-
      {{- .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end -}}
    format_overrides:
      - goos: windows
        formats: [zip]
    builds_info:
      group: root
      owner: root
      mtime: "{{ .CommitDate }}"
    files:
      - src: README.md
        info:
          owner: root
          group: root
          mtime: "{{ .CommitDate }}"
      - src: LICENSE.md
        info:
          owner: root
          group: root
          mtime: "{{ .CommitDate }}"

sboms:
  - artifacts: archive

signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    artifacts: checksum
    args:
      - sign-blob
      - "--bundle=${signature}"
      - "${artifact}"
      - --yes
```

**Key features:**
- Multi-platform builds (Linux, macOS, Windows)
- Multiple architectures (386, amd64, arm, arm64)
- Reproducible builds with mod_timestamp
- Changelog generation from conventional commits
- Archive creation with README/LICENSE
- SBOM (Software Bill of Materials) generation
- Cosign signing for supply chain security
- -trimpath for reproducible binaries
- CGO_ENABLED=0 for static binaries


### Makefile (optional but common)

```makefile
.PHONY: build test lint clean

build:
	go build -o bin/myapp ./cmd/myapp

test:
	go test -race -cover ./...

test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run

clean:
	rm -rf bin/ coverage.out coverage.html
```
