---
name: golang-code-review
description: Comprehensive Go code review following strict standards. Use when reviewing Go code, pull requests, checking compliance with coding standards, identifying anti-patterns, or assessing code quality. Enforces the run function pattern, dependency injection, no globals, proper error handling, black-box testing with 80% coverage, conventional commits, and identifies security/performance issues. Provides severity-based feedback (Critical, Major, Minor) with concrete examples and fixes.
---

# Golang Code Review

Perform thorough, standards-based code reviews of Go code with concrete, actionable feedback.

## Review Approach

This skill provides **systematic, multi-level reviews** with:

1. **Severity Classification**:
   - 🔴 **CRITICAL**: Must fix before merge (security, crashes, leaks)
   - 🟡 **MAJOR**: Should fix (standards violations, maintainability)
   - 🔵 **MINOR**: Nice to have (style, optimizations)
   - ℹ️ **INFO**: Suggestions only

2. **Concrete Examples**: Every issue includes:
   - Problematic code snippet
   - Clear explanation of the problem
   - Suggested fix with code

3. **Standards-Based**: Reviews against established Go standards:
   - run function pattern
   - No global commands/init()
   - Dependency injection
   - Interface design (small, defined where used)
   - Modern Go (1.21+)
   - Error handling (wrapping, guard clauses)
   - Black-box testing with 80% coverage
   - Conventional commits

## Quick Start

**For a full PR review:**
1. Read [references/review-checklist.md](references/review-checklist.md)
2. Apply checklist systematically
3. Use template from [references/review-templates.md](references/review-templates.md)
4. Provide severity-classified feedback

**For quick file/function review:**
- Use Quick Review Template from templates
- Focus on critical and major issues
- Provide code examples

**To identify anti-patterns:**
- Reference [references/anti-patterns.md](references/anti-patterns.md)
- Flag matches with appropriate severity

## Review Workflow

### Step 1: Understand the Change

- Read the PR/commit description
- Understand the intent and context
- Identify the scope (new feature, bug fix, refactor)

### Step 2: High-Level Review

Check architecture and design first:
- Does the approach make sense?
- Is the code organized logically?
- Are there better patterns available?
- Is the complexity appropriate?

### Step 3: Standards Compliance Check

Use the [review checklist](references/review-checklist.md) to check:

**Critical Standards** (🔴 if violated):
1. run function pattern in main
2. No goroutine leaks (clear shutdown)
3. No context in structs
4. Resources properly closed
5. Mutex not copied
6. time.After not in loops

**Major Standards** (🟡 if violated):
1. Dependency injection used
2. No global commands or init() for flags
3. Errors wrapped with fmt.Errorf("%w", err)
4. Guard clauses (no nested else)
5. Black-box testing (separate package)
6. No mixed pointer/value receivers
7. Context as first argument

**Minor Standards** (🔵 if violated):
1. Modern Go features (any, slices/maps, min/max)
2. Naming conventions (CamelCase, acronyms)
3. No Get prefix for getters
4. Slice capacity pre-allocation
5. strings.Builder for concatenation

### Step 4: Anti-Pattern Detection

Check [anti-patterns.md](references/anti-patterns.md) for common mistakes:
- God objects
- Premature abstraction
- Error shadowing
- Naked returns
- Goroutine leaks
- Ignored errors
- Resource leaks
- Mutex copying
- Empty interface abuse

### Step 5: Security & Performance

**Security checks**:
- Input validation
- SQL injection prevention
- No hardcoded secrets
- Error messages don't leak info
- Authentication/authorization present

**Performance checks**:
- No N+1 queries
- Goroutines managed properly
- Connection pooling configured
- Slice capacity considerations
- String concatenation in loops

### Step 6: Testing Review

**Requirements**:
- Minimum 80% code coverage (strict)
- Black-box testing (package_test, not package)
- Table-driven test pattern
- Using testify assertions
- Edge cases covered
- Error paths tested

**Check**:
```bash
go test -cover ./...
```

If coverage < 80%, this is a **🔴 CRITICAL** issue.

### Step 7: Commit Message Review

**Check against Conventional Commits**:
- Format: `<type>(<scope>): <description>`
- Valid types: fix, feat, build, chore, ci, docs, style, refactor, perf, test
- Clear, descriptive messages

Example violations:
```
❌ "fixed bug"           → 🟡 MAJOR: Not conventional format
❌ "updated code"        → 🟡 MAJOR: Vague message
✅ "fix(handler): resolve nil pointer in user lookup"
```

### Step 8: Provide Feedback

Use appropriate template from [review-templates.md](references/review-templates.md):
- Full Code Review Template (complete PRs)
- Quick Review Template (small changes)
- Focused Review Template (security/performance/testing)

**Feedback structure**:
1. Summary of what the code does
2. Issues by severity (Critical → Major → Minor)
3. Each issue includes:
   - File and line numbers
   - Code snippet showing the problem
   - Clear explanation
   - Suggested fix with code example
4. Positive observations (what's done well)
5. Overall recommendation (Approve/Request Changes/Comment)

## Review Examples

### Example 1: Critical Issue - Goroutine Leak

```markdown
**File: `internal/worker/worker.go`**
Lines 45-55:

```go
func (w *Worker) Start() {
    go func() {
        for {
            work := <-w.queue
            w.process(work)
        }
    }()
}
```

**🔴 CRITICAL:** Goroutine has no shutdown mechanism, will leak

**Impact:** Memory leak, resource exhaustion in production

**Fix:**
```go
func (w *Worker) Start(ctx context.Context) error {
    errCh := make(chan error, 1)
    
    go func() {
        for {
            select {
            case <-ctx.Done():
                errCh <- ctx.Err()
                return
            case work := <-w.queue:
                if err := w.process(ctx, work); err != nil {
                    errCh <- err
                    return
                }
            }
        }
    }()
    
    return <-errCh
}
```
```

### Example 2: Major Issue - No Dependency Injection

```markdown
**File: `internal/service/user.go`**
Lines 10-20:

```go
type UserService struct {}

func NewUserService() *UserService {
    db, _ := sql.Open("postgres", "...")  // Hardcoded connection
    return &UserService{db: db}
}
```

**🟡 MAJOR:** Dependencies created inside constructor, not injected

**Problems:**
- Untestable without database
- Violates dependency injection principle
- Hardcoded configuration

**Fix:**
```go
type UserService struct {
    db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
    return &UserService{db: db}
}
```

**Test example:**
```go
func TestUserService(t *testing.T) {
    mockDB := newMockDB()
    svc := NewUserService(mockDB)  // Easy to test
    // ...
}
```
```

### Example 3: Minor Issue - String Concatenation

```markdown
**File: `internal/formatter/format.go`**
Lines 30-35:

```go
func Format(items []string) string {
    result := ""
    for _, item := range items {
        result += item + "\n"
    }
    return result
}
```

**🔵 MINOR:** Inefficient string concatenation in loop

**Performance impact:** O(n²) due to string immutability

**Fix:**
```go
func Format(items []string) string {
    var builder strings.Builder
    for _, item := range items {
        builder.WriteString(item)
        builder.WriteString("\n")
    }
    return builder.String()
}
```
```

## Code Review Best Practices

### Be Constructive
- Focus on code, not the person
- Explain *why* something is an issue
- Provide concrete examples and fixes
- Acknowledge good patterns when you see them

### Be Thorough but Efficient
- Use the checklist to stay systematic
- Don't nitpick on style if automated tools handle it
- Focus on correctness, security, and maintainability first
- Group related issues together

### Be Clear
- Use severity levels consistently
- Include file names and line numbers
- Show both problematic code and suggested fix
- Explain the impact of each issue

### Be Realistic
- Not every minor issue needs fixing immediately
- Balance perfection with velocity
- Critical and major issues are blockers
- Minor issues can be follow-up tasks

## Standards Reference

This skill enforces the same standards as the golang-developer skill:

**Core patterns:**
- The run function pattern
- Dependency injection
- No global commands/init()
- Accept interfaces, return structs
- Modern Go (1.21+)
- Error wrapping and guard clauses
- Black-box testing with 80% coverage
- Conventional commits

**For detailed standards**, see:
- [Review Checklist](references/review-checklist.md) - Systematic checks
- [Anti-Patterns](references/anti-patterns.md) - Common mistakes to flag
- [Review Templates](references/review-templates.md) - Structured feedback formats

## Quick Reference

**Starting a review?**
→ Use [review-checklist.md](references/review-checklist.md)

**Found an unfamiliar pattern?**
→ Check [anti-patterns.md](references/anti-patterns.md)

**Need Go idioms reference?**
→ See [effective-go.md](references/effective-go.md) ⭐

**Understanding linter errors?**
→ See [linter-guide.md](references/linter-guide.md) 🔍

**Need review structure?**
→ Use templates from [review-templates.md](references/review-templates.md)

**Coverage check:**
```bash
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | grep total
```

**Linter check:**
```bash
golangci-lint run
```
