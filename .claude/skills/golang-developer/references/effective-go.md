# Effective Go Principles

Key principles from https://go.dev/doc/effective_go integrated with our coding standards.

## Formatting

**Use gofmt**: All code must be formatted with `gofmt` (or `go fmt`). Never work around it.

```bash
go fmt ./...
gofmt -w .
```

Key points:
- Tabs for indentation (gofmt default)
- No line length limit (wrap naturally with extra tab)
- No parentheses in control structures (if, for, switch)
- Operator precedence is shorter and clearer than C

## Commentary

**Doc comments**: Comments before top-level declarations (with no newline) document that declaration.

```go
// Package calculator provides arithmetic operations.
package calculator

// Add returns the sum of two integers.
// It handles both positive and negative numbers.
func Add(a, b int) int {
    return a + b
}
```

Rules:
- Start with the name of the element
- Use complete sentences
- Package comment goes before package declaration
- See [Go Doc Comments](https://go.dev/doc/comment) for details

## Names

### Package Names

- **Lowercase, single word**: No underscores or mixedCaps
- **Concise and evocative**: `bufio`, not `bufferedio`
- **Base name of directory**: `encoding/base64` → package `base64`
- **Avoid stuttering**: `bufio.Reader`, not `bufio.BufReader`

```go
// Good
import "encoding/base64"
decoder := base64.NewDecoder()

// Bad - stuttering
import "bufio"
reader := bufio.NewBufReader()  // Should be bufio.Reader
```

### No Get Prefix

```go
type Person struct {
    name string
}

// Good
func (p *Person) Name() string { return p.name }

// Bad
func (p *Person) GetName() string { return p.name }
```

### Interface Names

One-method interfaces: method name + `-er` suffix

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type Closer interface {
    Close() error
}
```

### MixedCaps

Use `MixedCaps` or `mixedCaps`, never underscores.

```go
// Good
type UserService struct {}
func parseUserInput() {}

// Bad
type User_Service struct {}
func parse_user_input() {}
```

## Control Structures

### Guard Clauses (Effective Go Pattern)

Eliminate error cases early, let successful flow run down the page.

```go
// Effective Go pattern - guard clauses
f, err := os.Open(name)
if err != nil {
    return err
}
defer f.Close()

d, err := f.Stat()
if err != nil {
    return err
}

codeUsing(f, d)
```

**Not this:**
```go
// Bad - nested else blocks
f, err := os.Open(name)
if err != nil {
    return err
} else {
    d, err := f.Stat()
    if err != nil {
        return err
    } else {
        codeUsing(f, d)
    }
}
```

### Redeclaration with :=

`:=` can redeclare variables in the same scope if:
- At least one new variable is created
- Existing variable gets a new value

```go
f, err := os.Open(name)
if err != nil {
    return err
}

d, err := f.Stat()  // err is redeclared, d is new
if err != nil {
    return err
}
```

### Switch Without Expression

Go idiom: use switch on `true` for if-else chains.

```go
// Effective Go pattern
func unhex(c byte) byte {
    switch {
    case '0' <= c && c <= '9':
        return c - '0'
    case 'a' <= c && c <= 'f':
        return c - 'a' + 10
    case 'A' <= c && c <= 'F':
        return c - 'A' + 10
    }
    return 0
}
```

## Functions

### Multiple Return Values

Return both value and error.

```go
// Good - multiple returns
func (file *File) Write(b []byte) (n int, err error) {
    // Write bytes, return count and any error
    return len(b), nil
}
```

### Named Result Parameters

Use named results for documentation, not for naked returns in our code.

```go
// Documentation value only
func nextInt(b []byte, pos int) (value, nextPos int) {
    // Clear what each int represents
}

// But still return explicitly (our standard)
func nextInt(b []byte, pos int) (value, nextPos int) {
    // ... logic ...
    return value, nextPos  // Explicit, not naked
}
```

**Note**: While Effective Go shows naked returns, our standard requires explicit returns for clarity.

### Defer

Defer runs immediately before function returns. Use for cleanup.

```go
func Contents(filename string) (string, error) {
    f, err := os.Open(filename)
    if err != nil {
        return "", err
    }
    defer f.Close()  // Guaranteed to run
    
    // ... use f ...
    return string(result), nil
}
```

**Defer facts**:
- Arguments evaluated when defer executes
- Deferred functions run in LIFO order
- Runs even if function panics
- Perfect for mutex.Unlock(), file.Close(), etc.

## Data Structures

### Zero Value Usefulness

Design types so zero value is useful without initialization.

```go
// Good - zero value is useful
type Buffer struct {
    buf []byte
}

var b Buffer
b.Write([]byte("hello"))  // Works! Zero value ready to use

// Good - sync.Mutex zero value is unlocked
var mu sync.Mutex
mu.Lock()  // Works immediately
```

### Composite Literals

Create and initialize in one expression.

```go
// Full initialization
return &File{
    fd:      fd,
    name:    name,
    dirinfo: nil,
    nepipe:  0,
}

// Partial initialization - rest get zero values
return &File{fd: fd, name: name}

// Zero value
return &File{}  // Equivalent to new(File)
```

### new vs make

**new(T)**:
- Allocates zeroed memory
- Returns `*T`
- For any type

**make(T, args)**:
- Initializes internal structure
- Returns `T` (not pointer)
- Only for slices, maps, channels

```go
// new - rarely used
p := new([]int)      // p is *[]int, *p is nil slice

// make - common
v := make([]int, 100) // v is []int, initialized slice

// Idiomatic
v := make([]int, 100)        // slice
m := make(map[string]int)    // map
ch := make(chan int, 10)     // buffered channel
```

### Slices

Slices are references to arrays. Changes visible through all references.

```go
// Pre-allocate capacity when size known
results := make([]Result, 0, len(items))
for _, item := range items {
    results = append(results, process(item))
}
```

**Two-dimensional slices**:
```go
// Allocate independently (flexible)
picture := make([][]uint8, YSize)
for i := range picture {
    picture[i] = make([]uint8, XSize)
}

// Single allocation (more efficient)
picture := make([][]uint8, YSize)
pixels := make([]uint8, XSize*YSize)
for i := range picture {
    picture[i], pixels = pixels[:XSize], pixels[XSize:]
}
```

### Maps

**Comma-ok idiom** for presence testing:

```go
seconds, ok := timeZone[tz]
if !ok {
    // tz not in map
}

// Or combined
if seconds, ok := timeZone[tz]; ok {
    return seconds
}

// Delete
delete(timeZone, "PDT")
```

## Methods

### Pointer vs Value Receivers

**Pointer receivers** when:
- Method modifies receiver
- Receiver is large struct (avoid copying)
- Consistency (if any method needs pointer, use pointer for all)

**Value receivers** when:
- Method doesn't modify receiver
- Receiver is small (int, small struct)
- Receiver is map, func, or chan (already reference types)

```go
// Pointer receiver - modifies state
func (c *Counter) Increment() {
    c.count++
}

// Value receiver - read-only
func (c Counter) Value() int {
    return c.count
}
```

**Our standard**: Be consistent. Use pointer receivers for all methods on a type if any method needs it.

## Interfaces

### Small Interfaces

Interface with one or two methods are most powerful and composable.

```go
// Good - small, focused interfaces
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

// Compose when needed
type ReadWriter interface {
    Reader
    Writer
}
```

### Type Assertions

**Comma-ok idiom** for safe type assertions:

```go
str, ok := value.(string)
if !ok {
    // value is not a string
}

// Or combined
if str, ok := value.(string); ok {
    return str
}
```

**Type switches**:
```go
switch v := value.(type) {
case string:
    // v is string
    fmt.Println(v)
case int:
    // v is int
    fmt.Println(v)
default:
    // unknown type
    fmt.Printf("unexpected type %T\n", v)
}
```

### Interface Generality

If type exists only to implement interface, don't export the type.

```go
// Good - only interface exported
package hash

type Hash interface {
    Write(p []byte) (n int, err error)
    Sum() []byte
}

func NewMD5() Hash {
    return &md5digest{}  // unexported implementation
}

type md5digest struct {
    // private fields
}
```

## The Blank Identifier

Use `_` to:
- Discard unwanted values in multiple assignment
- Import for side effects
- Silence unused variable/import errors during development

```go
// Discard index in range
for _, value := range array {
    sum += value
}

// Discard error (generally bad, but sometimes necessary)
_ = someFunc()  // Explicitly ignoring error

// Import for side effects (e.g., driver registration)
import _ "github.com/lib/pq"

// Compile-time interface check
var _ io.Reader = (*MyType)(nil)
```

## Embedding

Go favors composition over inheritance.

```go
// Embed to get methods automatically
type ReadWriter struct {
    *Reader  // Embedded - gets Read method
    *Writer  // Embedded - gets Write method
}

// Can override embedded methods
func (rw *ReadWriter) Write(p []byte) (n int, err error) {
    // Custom implementation
    return rw.Writer.Write(p)
}
```

## Concurrency Patterns

### Goroutines

Start with `go` keyword. Always know how it stops.

```go
// Good - clear shutdown
func (s *Service) Start(ctx context.Context) error {
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

### Channels

- Buffered vs unbuffered
- Use for signaling, not state
- Close to signal "no more values"

```go
// Buffered channel
ch := make(chan int, 100)

// Signal completion
done := make(chan struct{})
go func() {
    // work
    done <- struct{}{}  // Signal done
}()
<-done  // Wait for signal

// Range over channel (until closed)
for value := range ch {
    process(value)
}
```

## Errors

Not part of original Effective Go (2009) but essential:

**Wrap errors** with context:
```go
if err != nil {
    return fmt.Errorf("read config: %w", err)
}
```

**Check all errors**:
```go
// Good
if err := someFunc(); err != nil {
    return err
}

// Bad
someFunc()  // Ignoring error
_ = someFunc()  // Explicitly ignoring (rarely justified)
```

## Summary

Key Effective Go patterns to remember:
1. Format with gofmt always
2. Guard clauses for error handling
3. No Get prefix on getters
4. Small, focused interfaces
5. Defer for cleanup (Close, Unlock)
6. Zero values should be useful
7. Pointer receivers when modifying or large types
8. Comma-ok idiom for safe operations
9. Composition over inheritance (embedding)
10. Always know how goroutines stop
