# Linter Configuration Explained

This document explains the key linters in our `.golangci.yml` configuration and why they matter.

## Critical Linters

### bodyclose
**What**: Checks that HTTP response bodies are closed.

```go
// ❌ BAD - Response body not closed (memory leak)
resp, err := http.Get(url)
if err != nil {
    return err
}
// Missing: defer resp.Body.Close()
data, _ := io.ReadAll(resp.Body)

// ✅ GOOD
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()
data, err := io.ReadAll(resp.Body)
```

**Why**: Unclosed response bodies cause memory leaks and connection pool exhaustion.

---

### copyloopvar (Go 1.22+)
**What**: Detects loop variable capture issues.

```go
// ❌ BAD - All goroutines capture same variable
for _, item := range items {
    go func() {
        process(item)  // Bug: 'item' changes each iteration
    }()
}

// ✅ GOOD - Each goroutine gets its own copy
for _, item := range items {
    item := item  // Create loop-scoped copy
    go func() {
        process(item)
    }()
}

// ✅ BETTER (Go 1.22+) - Automatic per-iteration variables
for _, item := range items {
    go func() {
        process(item)  // Works correctly in Go 1.22+
    }()
}
```

**Why**: Prevents common goroutine bugs where all goroutines reference the same variable.

---

### depguard
**What**: Blocks usage of deprecated or discouraged packages.

Our config blocks:
- `github.com/pkg/errors` → Use stdlib `errors` and `fmt.Errorf("%w", err)` instead
- `math/rand` → Use `math/rand/v2` instead (better API, cryptographically secure by default)

```go
// ❌ BAD
import "github.com/pkg/errors"
err := errors.Wrap(err, "failed")

// ✅ GOOD
import "fmt"
err := fmt.Errorf("failed: %w", err)

// ❌ BAD
import "math/rand"
n := rand.Intn(100)

// ✅ GOOD
import "math/rand/v2"
n := rand.IntN(100)  // Note: IntN, not Intn
```

**Why**: Keeps dependencies modern and leverages stdlib improvements.

---

### forbidigo
**What**: Forbids specific function/package patterns.

Our config forbids: `ioutil.*`

```go
// ❌ BAD - ioutil is deprecated
import "io/ioutil"
data, err := ioutil.ReadFile(name)

// ✅ GOOD
import "os"
data, err := os.ReadFile(name)
```

**Why**: `ioutil` was deprecated in Go 1.16. Functions moved to `os` and `io` packages.

---

### gochecknoglobals
**What**: Enforces no package-level variables (with exceptions).

```go
// ❌ BAD - Mutable global state
var counter int

func Increment() {
    counter++  // Race condition in concurrent code
}

// ✅ GOOD - Pass state explicitly
type Counter struct {
    mu    sync.Mutex
    count int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}
```

**Exceptions allowed**:
- Constants
- Errors (`var ErrNotFound = errors.New("not found")`)
- Package-level init functions (sparingly)

**Why**: Global mutable state makes testing difficult and causes race conditions.

---

### goconst
**What**: Finds repeated strings/numbers that should be constants.

```go
// ❌ BAD - Magic strings repeated
if status == "active" {
    // ...
}
if record.Status == "active" {
    // ...
}

// ✅ GOOD
const StatusActive = "active"

if status == StatusActive {
    // ...
}
if record.Status == StatusActive {
    // ...
}
```

**Why**: Reduces typos and makes refactoring easier.

---

### goerr113
**What**: Checks that errors are wrapped with `%w` and not compared with `==`.

```go
// ❌ BAD - Error not wrapped
if err != nil {
    return fmt.Errorf("failed: %v", err)  // Using %v, not %w
}

// ❌ BAD - Comparing errors with ==
if err == io.EOF {  // Fragile
    // ...
}

// ✅ GOOD - Error wrapped
if err != nil {
    return fmt.Errorf("failed: %w", err)  // Using %w
}

// ✅ GOOD - Using errors.Is
if errors.Is(err, io.EOF) {
    // ...
}
```

**Why**: `%w` preserves error chain for `errors.Is()` and `errors.As()`.

---

### gosec
**What**: Security-focused static analysis.

Detects:
- SQL injection risks
- Command injection
- Weak crypto usage
- Insecure random number generation
- Path traversal vulnerabilities
- Hardcoded credentials

```go
// ❌ BAD - SQL injection risk
query := "SELECT * FROM users WHERE id = " + userInput
db.Exec(query)

// ✅ GOOD
query := "SELECT * FROM users WHERE id = ?"
db.Exec(query, userInput)

// ❌ BAD - Weak crypto
import "crypto/md5"
hash := md5.Sum(data)

// ✅ GOOD
import "crypto/sha256"
hash := sha256.Sum256(data)
```

**Why**: Catches common security vulnerabilities before production.

---

## Important Linters

### gocritic
**What**: Meta-linter with many checks for code quality.

Disabled checks:
- `appendAssign` - We allow `slice = append(slice, item)`

Catches:
- Inefficient operations
- Common mistakes
- Style issues
- Performance problems

---

### revive
**What**: Configurable linter with many rules.

Key rules enabled:
- `context-as-argument` - Context must be first parameter
- `error-naming` - Errors should be named `err` or end with `Error`
- `error-return` - Error should be last return value
- `error-strings` - Error strings shouldn't be capitalized
- `indent-error-flow` - Use guard clauses
- `superfluous-else` - Eliminate unnecessary else blocks
- `unexported-return` - Don't return unexported types from exported functions

```go
// ❌ BAD - Context not first
func Process(id string, ctx context.Context) error

// ✅ GOOD
func Process(ctx context.Context, id string) error

// ❌ BAD - Error string capitalized
return errors.New("Failed to connect")

// ✅ GOOD
return errors.New("failed to connect")

// ❌ BAD - Superfluous else
if err != nil {
    return err
} else {
    return process()
}

// ✅ GOOD
if err != nil {
    return err
}
return process()
```

---

### testifylint
**What**: Best practices for testify/assert usage.

```go
// ❌ BAD - Wrong assertion
assert.True(t, err != nil)

// ✅ GOOD
assert.Error(t, err)

// ❌ BAD - Wrong order
assert.Equal(t, actual, expected)

// ✅ GOOD - Expected first, actual second
assert.Equal(t, expected, actual)
```

---

### usetesting
**What**: Enforces using testing helpers instead of package-level functions.

```go
// ❌ BAD - Using package-level functions in tests
func TestExample(t *testing.T) {
    dir := os.TempDir()  // Not cleaned up
    os.Setenv("KEY", "value")  // Affects other tests
}

// ✅ GOOD - Using testing helpers
func TestExample(t *testing.T) {
    dir := t.TempDir()  // Automatically cleaned up
    t.Setenv("KEY", "value")  // Automatically restored
}
```

**Why**: Testing helpers provide automatic cleanup and isolation.

---

## Formatters

### gofumpt
**What**: Stricter version of `gofmt`.

Additional rules:
- Extra whitespace removed
- Consistent grouping of imports
- Consistent struct field alignment

```go
// gofmt allows this
import (
    "fmt"
    "os"

    "github.com/user/pkg"
)

// gofumpt requires this (single blank line)
import (
    "fmt"
    "os"

    "github.com/user/pkg"
)
```

---

### goimports
**What**: Automatically manages imports.

- Adds missing imports
- Removes unused imports
- Groups and sorts imports (stdlib, external, internal)

---

## Exclusions

### Test Files
Some linters are relaxed for test files:
- `noctx` - Tests don't always need context
- `perfsprint` - Performance less critical in tests

### Generated Code
Linters are relaxed for:
- Generated files (comment marker: `// Code generated`)
- `third_party/` directories
- `builtin/` directories
- `examples/` directories

---

## Running Linters

```bash
# Run all linters
golangci-lint run

# Run specific linter
golangci-lint run --enable-only=gosec

# Run with verbose output
golangci-lint run -v

# Run on specific files
golangci-lint run ./internal/...

# Auto-fix issues (where possible)
golangci-lint run --fix
```

---

## CI Integration

Add to `.github/workflows/lint.yml`:

```yaml
name: Lint
on: [push, pull_request]
jobs:
  golangci:
    name: lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
```

---

## Performance

The configuration uses:
- `version: "2"` - New faster configuration format
- Excludes generated/third-party code
- Caches results between runs

Typical run time: 5-15 seconds for medium projects.

---

## Customization

To disable a specific linter for one line:

```go
//nolint:gosec // Reason: this is test data
password := "hardcoded-test-password"
```

To disable for entire file:
```go
//go:build tools
// +build tools

package tools
```

**Important**: Always provide a reason when using `//nolint` directives.
