# Code Review Checklist

Use this checklist systematically when reviewing Go code. Mark violations with severity levels.

## Severity Levels

- 🔴 **CRITICAL**: Must fix before merge (security, data loss, crashes)
- 🟡 **MAJOR**: Should fix (performance issues, maintainability problems, standard violations)
- 🔵 **MINOR**: Nice to have (style preferences, optimization opportunities)
- ℹ️ **INFO**: Suggestions for improvement (no action required)

---

## 1. The run Function Pattern

**Check**: Does main follow the ultra-simple pattern?

```go
// ✅ CORRECT
func main() {
    ctx := context.Background()
    levelVar := &slog.LevelVar{}
    levelVar.Set(slog.LevelInfo)
    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: levelVar}))
    
    if err := run(ctx, os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr, logger, levelVar); err != nil {
        logger.Error("fatal error", slog.String("error", err.Error()))
        os.Exit(1)
    }
}

// ❌ WRONG - Logic in main
func main() {
    db, err := sql.Open("postgres", "...")
    if err != nil {
        log.Fatal(err)
    }
    // ... more logic
}
```

**Violations**:
- 🔴 Logic in main instead of run function
- 🔴 Missing dependency injection (args, getenv, stdin/stdout/stderr)
- 🟡 Missing structured logging setup
- 🟡 Missing dynamic log level (levelVar)

---

## 2. Global Variables & init()

**Check**: Are there package-level command variables or init() functions for CLI flags?

```go
// ❌ WRONG - Package-level command
var rootCmd = &cobra.Command{
    Use: "myapp",
}

func init() {
    rootCmd.Flags().StringVar(&config, "config", "", "config file")
}

// ✅ CORRECT - Constructor pattern
func NewRootCmd(logger *slog.Logger, levelVar *slog.LevelVar) *cobra.Command {
    var config string
    cmd := &cobra.Command{Use: "myapp"}
    cmd.Flags().StringVar(&config, "config", "", "config file")
    return cmd
}
```

**Violations**:
- 🔴 Package-level command variables
- 🔴 Using init() for flag registration
- 🟡 Global state that makes testing difficult

---

## 3. Dependency Injection

**Check**: Are dependencies injected through constructors?

```go
// ❌ WRONG - Creating dependencies internally
type Service struct {}

func NewService() *Service {
    db, _ := sql.Open("postgres", "...")  // Hard to test
    return &Service{db: db}
}

// ✅ CORRECT - Dependencies injected
type Service struct {
    db *sql.DB
}

func NewService(db *sql.DB) *Service {
    return &Service{db: db}
}
```

**Violations**:
- 🔴 Creating dependencies inside constructors
- 🟡 Hard-coded configuration values
- 🟡 Untestable code

---

## 4. Interfaces

**Check**: Are interfaces small, defined where used, and accepting interfaces/returning structs?

```go
// ❌ WRONG - Large interface, defined with implementation
package repository

type Repository interface {
    Get(ctx context.Context, id string) (*Item, error)
    List(ctx context.Context) ([]*Item, error)
    Save(ctx context.Context, item *Item) error
    Delete(ctx context.Context, id string) error
    Update(ctx context.Context, item *Item) error
    FindByName(ctx context.Context, name string) (*Item, error)
}

// ✅ CORRECT - Small interface, defined where used
package service

type ItemGetter interface {
    Get(ctx context.Context, id string) (*Item, error)
}

type Service struct {
    repo ItemGetter
}

func New(repo ItemGetter) *Service {  // Accept interface
    return &Service{repo: repo}       // Return struct
}
```

**Violations**:
- 🟡 Interface with >3 methods
- 🟡 Interface defined with implementation instead of consumer
- 🟡 Returning interface instead of struct

---

## 5. Error Handling

**Check**: Are errors wrapped, guard clauses used, and no panic in normal flow?

```go
// ❌ WRONG - Not wrapping errors, nested conditions
func Process(id string) error {
    item, err := get(id)
    if err == nil {
        if validate(item) == nil {
            if save(item) == nil {
                return nil
            } else {
                return save(item)
            }
        } else {
            return validate(item)
        }
    }
    return err
}

// ✅ CORRECT - Wrapped errors, guard clauses
func Process(ctx context.Context, id string) error {
    item, err := get(ctx, id)
    if err != nil {
        return fmt.Errorf("get item: %w", err)
    }
    
    if err := validate(item); err != nil {
        return fmt.Errorf("validate: %w", err)
    }
    
    if err := save(ctx, item); err != nil {
        return fmt.Errorf("save: %w", err)
    }
    
    return nil
}
```

**Violations**:
- 🔴 Using panic() in normal control flow
- 🟡 Not wrapping errors with fmt.Errorf("%w", err)
- 🟡 Nested else blocks instead of guard clauses
- 🟡 Swallowing errors (err != nil but not handled)

---

## 6. Naming Conventions

**Check**: Proper CamelCase, acronym handling, no Get prefix?

```go
// ❌ WRONG
type userService struct {}          // Should be unexported if internal
func (u *User) GetName() string {}  // No Get prefix
func ServeHttp() {}                 // Should be ServeHTTP
func ParseUrl() {}                  // Should be ParseURL

// ✅ CORRECT
type UserService struct {}          // Exported
func (u *User) Name() string {}     // No Get prefix
func ServeHTTP() {}                 // Acronym uppercase
func ParseURL() {}                  // Acronym uppercase
```

**Violations**:
- 🔵 Using Get prefix for getters
- 🔵 Incorrect acronym casing (Http vs HTTP, Url vs URL)
- 🔵 Inconsistent naming between exported/unexported

---

## 7. Modern Go Features (1.21+)

**Check**: Using modern Go idioms?

```go
// ❌ WRONG - Old patterns
var i interface{} = 42
result := math.Min(float64(a), float64(b))
sort.Strings(items)

// ✅ CORRECT - Modern Go 1.21+
var i any = 42
result := min(a, b)
slices.Sort(items)
```

**Violations**:
- 🔵 Using interface{} instead of any
- 🔵 Not using built-in min/max functions
- 🔵 Not using slices/maps packages

---

## 8. Concurrency

**Check**: Are APIs synchronous by default, context passed first, goroutine lifecycle clear?

```go
// ❌ WRONG
func (s *Service) Process(id string) {
    go func() {
        // No way to stop this goroutine
        // No context
        item := s.get(id)
        s.save(item)
    }()
}

// ✅ CORRECT
func (s *Service) Process(ctx context.Context, id string) error {
    errCh := make(chan error, 1)
    
    go func() {
        for {
            select {
            case <-ctx.Done():
                errCh <- ctx.Err()
                return
            case work := <-s.workQueue:
                if err := s.process(ctx, work); err != nil {
                    errCh <- err
                    return
                }
            }
        }
    }()
    
    return <-errCh
}
```

**Violations**:
- 🔴 Goroutine with no clear stop mechanism
- 🟡 Context not passed as first argument
- 🟡 API is async when it should be sync by default
- 🟡 Using channels for state instead of sync.Mutex

---

## 9. Testing

**Check**: Black-box tests, table-driven, 80% coverage?

```go
// ❌ WRONG - White-box testing
package service

func TestService(t *testing.T) {
    s := &Service{}
    s.internalMethod()  // Testing private methods
}

// ✅ CORRECT - Black-box testing
package service_test

import (
    "testing"
    "github.com/user/app/internal/service"
    "github.com/stretchr/testify/assert"
)

func TestService_PublicMethod(t *testing.T) {
    tests := []struct {
        name string
        input string
        want string
    }{
        {name: "case1", input: "a", want: "A"},
        {name: "case2", input: "b", want: "B"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := service.New()
            got := svc.PublicMethod(tt.input)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

**Violations**:
- 🔴 Tests in same package (not black-box)
- 🔴 Coverage below 80%
- 🟡 Not using table-driven test pattern
- 🟡 Not using testify assertions

---

## 10. HTTP Patterns (Mat Ryer)

**Check**: Server struct, route handlers return closures, proper middleware?

```go
// ❌ WRONG - Handlers directly registered
http.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
    // No server context, no logger
})

// ✅ CORRECT - Mat Ryer pattern
type server struct {
    db     *sql.DB
    router *http.ServeMux
    logger *slog.Logger
}

func (s *server) routes() {
    s.router.HandleFunc("GET /items", s.handleItems())
}

func (s *server) handleItems() http.HandlerFunc {
    // One-time setup
    type response struct {
        Items []Item `json:"items"`
    }
    
    return func(w http.ResponseWriter, r *http.Request) {
        // Per-request handling
        s.respond(w, r, response{}, http.StatusOK)
    }
}
```

**Violations**:
- 🟡 Not using server struct pattern
- 🟡 Handlers don't return closures
- 🟡 Missing request/response helper methods
- 🔵 Not using Go 1.22+ routing (GET /path syntax)

---

## 11. Logging

**Check**: Using log/slog with structured logging?

```go
// ❌ WRONG
log.Println("Processing item", id)
fmt.Printf("Error: %v\n", err)

// ✅ CORRECT
logger.Info("processing item", slog.String("id", id))
logger.Error("operation failed", slog.String("error", err.Error()))
```

**Violations**:
- 🟡 Using log.Println or fmt.Printf instead of slog
- 🟡 Unstructured logging
- 🔵 Not using appropriate log levels

---

## 12. Dependencies

**Check**: Are dependencies well-researched, maintained, and necessary?

**Questions to ask**:
- Is this the de facto standard library for this use case?
- Is it actively maintained?
- Could stdlib be used instead?
- Is the API ergonomic?

**Violations**:
- 🟡 Using unmaintained dependencies
- 🟡 Adding dependency for simple functionality stdlib provides
- 🔵 Not using pre-approved libraries (cobra, viper, testify, prometheus)

---

## 13. Code Organization

**Check**: Proper project structure, internal/ usage?

**Violations**:
- 🟡 Business logic in main package
- 🟡 Not using internal/ for private packages
- 🟡 Circular dependencies
- 🔵 Deep nesting (>3-4 levels)

---

## 14. Documentation

**Check**: Exported items documented, package doc comments?

```go
// ❌ WRONG - Missing documentation
type Service struct {}

func NewService() *Service {}

// ✅ CORRECT
// Package service provides business logic for managing items.
package service

// Service handles item operations with database access.
type Service struct {}

// New creates a new Service with the given database connection.
func New(db *sql.DB) *Service {}
```

**Violations**:
- 🔵 Missing package documentation
- 🔵 Exported types/functions without doc comments
- 🔵 Doc comments don't start with item name

---

## 15. Effective Go Idioms

**Check**: Following Effective Go patterns?

```go
// ❌ WRONG - Not using guard clauses
func Process() error {
    if condition {
        if anotherCondition {
            return doWork()
        } else {
            return errors.New("condition failed")
        }
    }
    return nil
}

// ✅ CORRECT - Guard clauses (Effective Go pattern)
func Process(ctx context.Context) error {
    if !condition {
        return nil
    }
    
    if !anotherCondition {
        return errors.New("condition failed")
    }
    
    return doWork(ctx)
}
```

```go
// ❌ WRONG - Not using defer
func ReadFile(name string) ([]byte, error) {
    f, err := os.Open(name)
    if err != nil {
        return nil, err
    }
    data, err := io.ReadAll(f)
    f.Close()  // Easy to miss in complex functions
    return data, err
}

// ✅ CORRECT - Defer pattern (Effective Go)
func ReadFile(name string) ([]byte, error) {
    f, err := os.Open(name)
    if err != nil {
        return nil, err
    }
    defer f.Close()  // Guaranteed to run
    
    return io.ReadAll(f)
}
```

**Violations**:
- 🟡 Not using guard clauses (nested else blocks)
- 🟡 Not using defer for cleanup
- 🔵 Package names not lowercase single word
- 🔵 Using Get prefix on getters
- 🔵 Not using comma-ok idiom for type assertions
- 🔵 Zero value not useful (requires explicit init)

For detailed Effective Go patterns, see [effective-go.md](effective-go.md).

---

## 16. Commit Messages

**Check**: Following Conventional Commits?

```
// ❌ WRONG
"fixed bug"
"updated code"
"changes"

// ✅ CORRECT
"fix(handler): resolve nil pointer in user lookup"
"feat(api): add authentication middleware"
"refactor(service): extract validation logic"
```

**Violations**:
- 🟡 Not following conventional commit format
- 🟡 Vague commit messages
- 🔵 Wrong type (fix vs feat vs refactor)
