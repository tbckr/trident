# Common Anti-Patterns in Go

Recognize and flag these common mistakes in code reviews.

## 1. God Objects

**Problem**: One struct/package doing too much.

```go
// ❌ ANTI-PATTERN
type Service struct {
    db *sql.DB
    cache *redis.Client
    queue *sqs.Queue
    email *smtp.Client
    logger *slog.Logger
}

func (s *Service) CreateUser() {}
func (s *Service) SendEmail() {}
func (s *Service) ProcessQueue() {}
func (s *Service) CacheData() {}
// ... 50 more methods
```

**Solution**: Split into focused services with single responsibilities.

```go
// ✅ BETTER
type UserService struct {
    repo UserRepository
    notifier Notifier
}

type EmailService struct {
    client *smtp.Client
}

type CacheService struct {
    client *redis.Client
}
```

**Review Flag**: 🟡 MAJOR - Service/struct has >10 methods or >5 dependencies

---

## 2. Premature Abstraction

**Problem**: Creating interfaces before they're needed.

```go
// ❌ ANTI-PATTERN - Interface with one implementation
type UserRepository interface {
    Get(id string) (*User, error)
    Save(user *User) error
}

type postgresUserRepository struct {}  // Only implementation

// Used nowhere else, just adds complexity
```

**Solution**: Start with concrete types. Add interfaces when you need them.

```go
// ✅ BETTER - Start concrete
type UserRepository struct {
    db *sql.DB
}

func (r *UserRepository) Get(ctx context.Context, id string) (*User, error) {}

// Add interface later when needed for testing or multiple implementations
```

**Review Flag**: 🔵 MINOR - Interface with single implementation and no clear future need

---

## 3. Error Shadowing

**Problem**: Declaring err multiple times, shadowing earlier errors.

```go
// ❌ ANTI-PATTERN
func Process() error {
    item, err := getItem()
    if err != nil {
        return err
    }
    
    if item.Valid {
        result, err := process(item)  // Shadows outer err
        if err != nil {
            return err
        }
        return save(result)
    }
    
    // This err might be from process(), not getItem()
    return err  
}
```

**Solution**: Use different variable names or immediate handling.

```go
// ✅ BETTER
func Process(ctx context.Context) error {
    item, err := getItem(ctx)
    if err != nil {
        return fmt.Errorf("get item: %w", err)
    }
    
    if !item.Valid {
        return nil
    }
    
    result, err := process(ctx, item)
    if err != nil {
        return fmt.Errorf("process: %w", err)
    }
    
    if err := save(ctx, result); err != nil {
        return fmt.Errorf("save: %w", err)
    }
    
    return nil
}
```

**Review Flag**: 🔴 CRITICAL - Variable shadowing that could mask errors

---

## 4. Naked Returns

**Problem**: Using named return values with bare return statements.

```go
// ❌ ANTI-PATTERN
func Get(id string) (user *User, err error) {
    user, err = db.Query(id)
    if err != nil {
        return  // Which values? Unclear!
    }
    return  // Is user set? Have to trace logic
}
```

**Solution**: Always return explicitly.

```go
// ✅ BETTER
func Get(ctx context.Context, id string) (*User, error) {
    user, err := db.Query(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("query: %w", err)
    }
    return user, nil
}
```

**Review Flag**: 🟡 MAJOR - Named return values with naked returns

---

## 5. Context in Struct

**Problem**: Storing context in struct fields.

```go
// ❌ ANTI-PATTERN
type Service struct {
    ctx context.Context  // NEVER do this
    db  *sql.DB
}

func NewService(ctx context.Context, db *sql.DB) *Service {
    return &Service{ctx: ctx, db: db}
}
```

**Solution**: Pass context as first argument to methods.

```go
// ✅ BETTER
type Service struct {
    db *sql.DB
}

func New(db *sql.DB) *Service {
    return &Service{db: db}
}

func (s *Service) Process(ctx context.Context, id string) error {
    // Use ctx parameter, not struct field
}
```

**Review Flag**: 🔴 CRITICAL - Context stored in struct

---

## 6. Goroutine Leaks

**Problem**: Starting goroutines with no way to stop them.

```go
// ❌ ANTI-PATTERN
func (s *Service) Start() {
    go func() {
        for {
            work := <-s.queue  // Will block forever
            s.process(work)
        }
    }()
}
```

**Solution**: Always provide a way to stop goroutines.

```go
// ✅ BETTER
func (s *Service) Start(ctx context.Context) error {
    errCh := make(chan error, 1)
    
    go func() {
        for {
            select {
            case <-ctx.Done():
                errCh <- ctx.Err()
                return
            case work := <-s.queue:
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

**Review Flag**: 🔴 CRITICAL - Goroutine with no clear shutdown path

---

## 7. Ignoring Errors

**Problem**: Using _ to ignore errors that should be handled.

```go
// ❌ ANTI-PATTERN
func Save(user *User) {
    json.Marshal(user)  // Ignoring error
    _ = db.Save(user)   // Explicitly ignoring
    defer file.Close()  // Ignoring error
}
```

**Solution**: Handle all errors appropriately.

```go
// ✅ BETTER
func Save(ctx context.Context, user *User) error {
    data, err := json.Marshal(user)
    if err != nil {
        return fmt.Errorf("marshal: %w", err)
    }
    
    if err := db.Save(ctx, user); err != nil {
        return fmt.Errorf("save: %w", err)
    }
    
    return nil
}
```

**Review Flag**: 🟡 MAJOR - Ignored errors (especially defer with error return)

---

## 8. Pointer Receivers Inconsistency

**Problem**: Mixing value and pointer receivers on the same type.

```go
// ❌ ANTI-PATTERN
type Counter struct {
    count int
}

func (c Counter) Increment() {
    c.count++  // Doesn't work, modifying copy
}

func (c *Counter) Value() int {  // Mixed receiver types
    return c.count
}
```

**Solution**: Be consistent - use pointer receivers for mutability.

```go
// ✅ BETTER
type Counter struct {
    count int
}

func (c *Counter) Increment() {
    c.count++
}

func (c *Counter) Value() int {
    return c.count
}
```

**Review Flag**: 🟡 MAJOR - Mixed value and pointer receivers on same type

---

## 9. Empty Interface Abuse

**Problem**: Using any/interface{} when specific types should be used.

```go
// ❌ ANTI-PATTERN
func Process(data any) any {
    // Type assertions everywhere
    if s, ok := data.(string); ok {
        return processString(s)
    }
    if i, ok := data.(int); ok {
        return processInt(i)
    }
    return nil
}
```

**Solution**: Use specific types or generics (Go 1.18+).

```go
// ✅ BETTER - Specific types
func ProcessString(s string) string { ... }
func ProcessInt(i int) int { ... }

// Or with generics
func Process[T any](data T) T { ... }
```

**Review Flag**: 🟡 MAJOR - Excessive use of any/interface{} instead of specific types

---

## 10. Not Closing Resources

**Problem**: Forgetting to close files, connections, response bodies.

```go
// ❌ ANTI-PATTERN
func Read(filename string) ([]byte, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    return io.ReadAll(file)  // File never closed
}
```

**Solution**: Always defer Close() immediately after opening.

```go
// ✅ BETTER
func Read(filename string) ([]byte, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, fmt.Errorf("open: %w", err)
    }
    defer file.Close()
    
    data, err := io.ReadAll(file)
    if err != nil {
        return nil, fmt.Errorf("read: %w", err)
    }
    
    return data, nil
}
```

**Review Flag**: 🔴 CRITICAL - Resource opened without defer Close()

---

## 11. Mutex Copy

**Problem**: Copying structs that contain mutexes.

```go
// ❌ ANTI-PATTERN
type Counter struct {
    mu    sync.Mutex
    count int
}

func process(c Counter) {  // Copies mutex!
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}
```

**Solution**: Always pass structs with mutexes as pointers.

```go
// ✅ BETTER
type Counter struct {
    mu    sync.Mutex
    count int
}

func process(c *Counter) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}
```

**Review Flag**: 🔴 CRITICAL - Struct with sync types passed by value

---

## 12. Time.After in Loops

**Problem**: Using time.After in loops creates memory leaks.

```go
// ❌ ANTI-PATTERN
for {
    select {
    case msg := <-messages:
        process(msg)
    case <-time.After(5 * time.Second):  // Creates new timer each iteration
        timeout()
    }
}
```

**Solution**: Use time.NewTimer and reset it.

```go
// ✅ BETTER
timer := time.NewTimer(5 * time.Second)
defer timer.Stop()

for {
    select {
    case msg := <-messages:
        if !timer.Stop() {
            <-timer.C
        }
        timer.Reset(5 * time.Second)
        process(msg)
    case <-timer.C:
        timeout()
        timer.Reset(5 * time.Second)
    }
}
```

**Review Flag**: 🔴 CRITICAL - time.After() used inside loop

---

## 13. Slicing Without Capacity

**Problem**: Not pre-allocating slice capacity when size is known.

```go
// ❌ ANTI-PATTERN
func process(items []Item) []Result {
    var results []Result  // Will grow multiple times
    for _, item := range items {
        results = append(results, transform(item))
    }
    return results
}
```

**Solution**: Pre-allocate with make when size is known.

```go
// ✅ BETTER
func process(items []Item) []Result {
    results := make([]Result, 0, len(items))
    for _, item := range items {
        results = append(results, transform(item))
    }
    return results
}
```

**Review Flag**: 🔵 MINOR - Slice without capacity pre-allocation

---

## 14. String Concatenation in Loops

**Problem**: Using + for string concatenation in loops.

```go
// ❌ ANTI-PATTERN
func build(items []string) string {
    result := ""
    for _, item := range items {
        result += item + "\n"  // Allocates new string each time
    }
    return result
}
```

**Solution**: Use strings.Builder for efficiency.

```go
// ✅ BETTER
func build(items []string) string {
    var builder strings.Builder
    for _, item := range items {
        builder.WriteString(item)
        builder.WriteString("\n")
    }
    return builder.String()
}
```

**Review Flag**: 🟡 MAJOR - String concatenation with + in loop

---

## 15. Not Using Constant for Magic Numbers

**Problem**: Magic numbers/strings scattered throughout code.

```go
// ❌ ANTI-PATTERN
if status == 200 {
    // ...
}
if role == "admin" {
    // ...
}
```

**Solution**: Define constants.

```go
// ✅ BETTER
const (
    StatusOK    = 200
    RoleAdmin   = "admin"
)

if status == StatusOK {
    // ...
}
if role == RoleAdmin {
    // ...
}
```

**Review Flag**: 🔵 MINOR - Magic numbers/strings not extracted to constants
