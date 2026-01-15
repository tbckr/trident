---
name: golang-developer
description: Professional Go development with strict adherence to best practices, idiomatic patterns, and modern standards (Go 1.21+). Use when writing Go code, setting up Go projects, implementing CLI applications with Cobra, building HTTP services following Mat Ryer patterns, writing tests with testify, or any Go development task. Enforces the run function pattern, dependency injection, black-box testing, 80% code coverage, conventional commits, and uses standard libraries (cobra, viper, slog, testify, prometheus/client_golang).
---

# Golang Developer

Write professional, idiomatic Go code following modern best practices and strict standards.

## Core Standards

**ALWAYS** follow these principles:

1. **The run Function Pattern**: main must be ultra-simple (context, logging, run, exit)
2. **Injection & Environment Control**: Pass args, getenv, stdin/out/err, logger, levelVar to run
3. **No Global Commands**: NEVER use package-level command variables or init() for flags
4. **Command Constructors**: Use constructors like NewRootCmd(logger, levelVar, ...)
5. **Accept Interfaces, Return Structs**: Define small interfaces (1-3 methods) where used
6. **Modern Go (1.21+)**: Use any, slices/maps packages, min/max, log/slog
7. **Error Handling**: Wrap with fmt.Errorf("%w", err), use guard clauses, never panic for normal flow
8. **Black-Box Testing**: Tests in separate test package (pkg_test), table-driven, 80% coverage mandatory
9. **Conventional Commits**: Strict adherence (fix:, feat:, build:, chore:, ci:, docs:, style:, refactor:, perf:, test:)
10. **Tooling**: Must pass golangci-lint v2, use goreleaser v2 for releases

## Standard Libraries

Use these pre-approved libraries:
- CLI: `github.com/spf13/cobra`
- Config: `github.com/spf13/viper`
- Logging: `log/slog` (stdlib)
- Testing: `github.com/stretchr/testify`
- Metrics: `github.com/prometheus/client_golang`

For HTTP: Use stdlib `net/http` ServeMux with Go 1.22+ routing enhancements.

## Workflow Decision Tree

Choose your path based on the task:

**Creating a new project?**
→ Go to [Project Setup](#project-setup)

**Writing Go code?**
→ Read [references/coding-standards.md](references/coding-standards.md) for detailed standards
→ For HTTP services: Read [references/http-patterns.md](references/http-patterns.md)
→ For CLI apps: Read [references/cli-patterns.md](references/cli-patterns.md)

**Writing tests?**
→ Read [references/testing-patterns.md](references/testing-patterns.md)

**Need project structure guidance?**
→ Read [references/project-structure.md](references/project-structure.md)

## Project Setup

### 1. Initialize Module

```bash
go mod init github.com/user/myapp
```

### 2. Choose Application Type

**CLI Application:**
```bash
mkdir -p cmd/myapp internal/cli
```
Copy templates from `assets/templates/cli/`:
- `main.go` → `cmd/myapp/main.go`
- `root.go` → `internal/cli/root.go`

**HTTP Application:**
```bash
mkdir -p cmd/myapp internal/server templates static/{css,js}
```
Copy templates from `assets/templates/http/`:
- `main.go` → `cmd/myapp/main.go`
- `server.go` → `internal/server/server.go`
- `handlers.go` → `internal/server/handlers.go`

### 3. Install Dependencies

```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get github.com/stretchr/testify@latest
go get github.com/prometheus/client_golang@latest
```

### 4. Setup Tooling

Copy production-ready configs from `assets/configs/`:
- `.golangci.yml` → project root
- `.goreleaser.yml` → project root

Or create `.golangci.yml` manually (see [references/project-structure.md](references/project-structure.md) for complete config):
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
    # ... see full config in references/project-structure.md
```

Create `.goreleaser.yml` (see [references/project-structure.md](references/project-structure.md) for complete config).

### 5. Update Templates

Replace placeholder imports (`github.com/user/myapp`) with your actual module path.

## Code Writing Guidelines

### Main Function Pattern

**ALWAYS** use this exact pattern:

```go
func main() {
    ctx := context.Background()
    
    levelVar := &slog.LevelVar{}
    levelVar.Set(slog.LevelInfo)
    
    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: levelVar,
    }))
    
    if err := run(ctx, os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr, logger, levelVar); err != nil {
        logger.Error("fatal error", slog.String("error", err.Error()))
        os.Exit(1)
    }
}

func run(
    ctx context.Context,
    args []string,
    getenv func(string) string,
    stdin io.Reader,
    stdout, stderr io.Writer,
    logger *slog.Logger,
    levelVar *slog.LevelVar,
) error {
    // All application logic here
    return nil
}
```

### CLI Commands - NO GLOBALS

**NEVER** do this:
```go
var rootCmd = &cobra.Command{} // WRONG - package-level variable

func init() {
    rootCmd.Flags().StringVar(...) // WRONG - init function
}
```

**ALWAYS** use constructors:
```go
func NewRootCmd(logger *slog.Logger, levelVar *slog.LevelVar) *cobra.Command {
    cmd := &cobra.Command{
        Use: "myapp",
    }
    cmd.AddCommand(NewServeCmd(logger))
    return cmd
}
```

### HTTP Services - Mat Ryer Pattern

Structure servers as:

```go
type server struct {
    db     *sql.DB
    router *http.ServeMux
    logger *slog.Logger
}

func newServer(db *sql.DB, logger *slog.Logger) *server {
    s := &server{
        db:     db,
        router: http.NewServeMux(),
        logger: logger,
    }
    s.routes()
    return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.router.ServeHTTP(w, r)
}

func (s *server) routes() {
    s.router.HandleFunc("GET /api/items", s.handleItemsList())
    s.router.HandleFunc("POST /api/items", s.handleItemsCreate())
}

func (s *server) handleItemsList() http.HandlerFunc {
    // One-time setup
    type response struct {
        Items []Item `json:"items"`
    }
    
    return func(w http.ResponseWriter, r *http.Request) {
        // Per-request handling
        items, err := s.getItems(r.Context())
        if err != nil {
            s.respondError(w, r, err, http.StatusInternalServerError)
            return
        }
        s.respond(w, r, response{Items: items}, http.StatusOK)
    }
}
```

### Error Handling

Use guard clauses and wrap errors:

```go
func ProcessItem(ctx context.Context, id string) error {
    item, err := getItem(ctx, id)
    if err != nil {
        return fmt.Errorf("get item: %w", err)
    }
    
    if err := validate(item); err != nil {
        return fmt.Errorf("validate item: %w", err)
    }
    
    if err := save(ctx, item); err != nil {
        return fmt.Errorf("save item: %w", err)
    }
    
    return nil
}
```

## Testing Requirements

**Minimum 80% coverage is mandatory.**

### Black-Box Testing Pattern

```go
// In file: internal/service/service.go
package service

type Service struct {
    repo Repository
}

func New(repo Repository) *Service {
    return &Service{repo: repo}
}
```

```go
// In file: internal/service/service_test.go
package service_test  // Note: service_test, not service

import (
    "testing"
    "github.com/user/myapp/internal/service"
    "github.com/stretchr/testify/assert"
)

func TestService_Method(t *testing.T) {
    // Can only use exported API
    svc := service.New(mockRepo)
    
    result, err := svc.Method()
    assert.NoError(t, err)
}
```

### Table-Driven Tests

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a        int
        b        int
        expected int
    }{
        {name: "positive", a: 2, b: 3, expected: 5},
        {name: "negative", a: -2, b: -3, expected: -5},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Coverage Check

```bash
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | grep total
```

Coverage must be ≥80%.

## Commit Message Format

**Strictly follow Conventional Commits:**

```
<type>(<scope>): <description>

[optional body]
```

**Types**: fix, feat, build, chore, ci, docs, style, refactor, perf, test

**Examples**:
```
feat(api): add user authentication endpoint

Implement JWT-based authentication with refresh tokens.
```

```
fix(database): resolve connection pool leak

Close connections properly in error paths.
```

## Quick Reference

**Need detailed patterns?** Read the appropriate reference file:
- Effective Go principles → [references/effective-go.md](references/effective-go.md) ⭐ Start here
- Coding standards → [references/coding-standards.md](references/coding-standards.md)
- Linter explanations → [references/linter-guide.md](references/linter-guide.md) 🔍
- HTTP patterns → [references/http-patterns.md](references/http-patterns.md)
- CLI patterns → [references/cli-patterns.md](references/cli-patterns.md)
- Testing patterns → [references/testing-patterns.md](references/testing-patterns.md)
- Project structure → [references/project-structure.md](references/project-structure.md)

**Starting templates** in `assets/templates/`:
- CLI: `cli/main.go`, `cli/root.go`
- HTTP: `http/main.go`, `http/server.go`, `http/handlers.go`

**Production configs** in `assets/configs/`:
- `.golangci.yml` - Comprehensive linter configuration
- `.goreleaser.yml` - Multi-platform release automation
