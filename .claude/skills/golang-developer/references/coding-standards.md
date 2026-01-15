# Coding Standards

## The run Function Pattern

Main must be ultra-simple. It initializes context, dynamic logging, calls run, and handles the final exit.

```go
package main

import (
    "context"
    "io"
    "log/slog"
    "os"
)

func main() {
    ctx := context.Background()
    
    // Setup structured logging with dynamic level
    levelVar := &slog.LevelVar{}
    levelVar.Set(slog.LevelInfo)
    
    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: levelVar,
    }))
    
    // Call run with all dependencies injected
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
    // All application logic goes here
    // This function is testable because all dependencies are injected
    return nil
}
```

## Dependency Research

Before adding any dependency, research the "de facto" standard:

1. **Search**: "golang <use-case> library"
2. **Evaluate**: Check GitHub stars, maintenance status, community adoption
3. **Prioritize**: Maintenance > Community adoption > API ergonomics
4. **Standard library first**: Prefer stdlib when it meets requirements

### Pre-approved Libraries

These are the standard libraries for this project:

- CLI: `github.com/spf13/cobra`
- Config: `github.com/spf13/viper`
- Logging: `log/slog` (stdlib)
- Testing: `github.com/stretchr/testify`
- Metrics: `github.com/prometheus/client_golang`

## Naming Conventions

```go
// Exported: CamelCase
type UserService struct {}
func NewUserService() *UserService {}

// Unexported: camelCase
type userRepository struct {}
func newUserRepository() *userRepository {}

// Keep acronyms uppercase
func ServeHTTP(w http.ResponseWriter, r *http.Request) {}
func ParseURL(s string) (*URL, error) {}

// No Get prefix for getters
type Person struct {
    name string
}

// Good
func (p *Person) Name() string { return p.name }

// Bad
func (p *Person) GetName() string { return p.name }
```

## Interfaces

Accept interfaces, return structs. Keep interfaces small (1-3 methods). Define them where used.

```go
// Bad: Interface defined with implementation
package repository

type Repository interface {
    Get(ctx context.Context, id string) (*Item, error)
    List(ctx context.Context) ([]*Item, error)
    Save(ctx context.Context, item *Item) error
    Delete(ctx context.Context, id string) error
}

type postgresRepository struct {}

func (r *postgresRepository) Get(...) {}
func (r *postgresRepository) List(...) {}
func (r *postgresRepository) Save(...) {}
func (r *postgresRepository) Delete(...) {}
```

```go
// Good: Interface defined where used
package service

// Small, focused interface
type ItemGetter interface {
    Get(ctx context.Context, id string) (*Item, error)
}

type Service struct {
    repo ItemGetter  // Accepts interface
}

func New(repo ItemGetter) *Service {  // Accepts interface
    return &Service{repo: repo}        // Returns struct
}

func (s *Service) GetItem(ctx context.Context, id string) (*Item, error) {
    return s.repo.Get(ctx, id)
}
```

## Modern Go Features (1.21+)

Use modern Go features:

```go
// Use 'any' instead of 'interface{}'
func Marshal(v any) ([]byte, error) {}

// Use slices package
import "slices"
items := []int{3, 1, 2}
slices.Sort(items)
if slices.Contains(items, 2) { }

// Use maps package
import "maps"
m := map[string]int{"a": 1, "b": 2}
clone := maps.Clone(m)

// Use min/max
result := min(a, b)
result := max(a, b)

// Use log/slog for structured logging
import "log/slog"
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("processing", slog.String("id", id), slog.Int("count", count))
```

## Error Handling

Errors are values. Wrap them with fmt.Errorf("%w", err). Use guard clauses to avoid nested else blocks.

```go
// Good: Guard clauses
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

// Bad: Nested else blocks
func ProcessItem(ctx context.Context, id string) error {
    item, err := getItem(ctx, id)
    if err == nil {
        if err := validate(item); err == nil {
            if err := save(ctx, item); err == nil {
                return nil
            } else {
                return err
            }
        } else {
            return err
        }
    } else {
        return err
    }
}
```

Never panic for normal control flow:

```go
// Good: Return error
func ParseConfig(data []byte) (*Config, error) {
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }
    return &cfg, nil
}

// Bad: Panic
func ParseConfig(data []byte) *Config {
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        panic(err)  // Never do this
    }
    return &cfg
}
```

## Concurrency

Keep APIs synchronous by default. Pass context.Context as the first argument. Always know how a goroutine stops.

```go
// Good: Synchronous API
func (s *Service) GetItems(ctx context.Context) ([]*Item, error) {
    // Implementation
    return items, nil
}

// Context as first argument
func (s *Service) Process(ctx context.Context, item *Item) error {
    // Check context
    if ctx.Err() != nil {
        return ctx.Err()
    }
    
    // Implementation
    return nil
}

// Always know how goroutine stops
func (s *Service) Start(ctx context.Context) error {
    errCh := make(chan error, 1)
    
    go func() {
        // Worker goroutine
        for {
            select {
            case <-ctx.Done():
                errCh <- ctx.Err()
                return
            case work := <-s.workQueue:
                if err := s.process(work); err != nil {
                    errCh <- err
                    return
                }
            }
        }
    }()
    
    return <-errCh
}
```

Use sync.Mutex for state, channels for signaling:

```go
// Mutex for protecting state
type Counter struct {
    mu    sync.Mutex
    count int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

// Channels for signaling
func Worker(ctx context.Context, work <-chan Task, done chan<- struct{}) {
    for {
        select {
        case <-ctx.Done():
            done <- struct{}{}
            return
        case task := <-work:
            process(task)
        }
    }
}
```

## Conventional Commits

Strictly follow the Conventional Commits specification.

**Format**: `<type>(<scope>): <description>`

**Allowed Types**:
- `fix:` - Bug fixes
- `feat:` - New features
- `build:` - Build system changes
- `chore:` - Maintenance tasks
- `ci:` - CI configuration
- `docs:` - Documentation
- `style:` - Code style (formatting)
- `refactor:` - Code refactoring
- `perf:` - Performance improvements
- `test:` - Adding/modifying tests

**Examples**:

```
feat(api): add user authentication endpoint

Implement JWT-based authentication with refresh tokens.
Includes middleware for protected routes.
```

```
fix(database): resolve connection pool leak

Close connections properly in error paths.
Add connection timeout configuration.
```

```
refactor(handler): extract validation logic

Move validation to separate package for reusability.
```

```
test(service): increase coverage to 85%

Add table-driven tests for edge cases.
```
