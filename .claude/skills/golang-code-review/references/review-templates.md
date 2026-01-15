# Review Templates

Use these templates for consistent code reviews.

## Full Code Review Template

Use for reviewing complete PRs or substantial changes.

```markdown
# Code Review: [PR Title/Description]

## Summary
[Brief overview of what this PR does]

## Review Results

### 🔴 CRITICAL Issues (Must Fix)
[List critical issues that must be fixed before merge]
- [ ] Issue 1: Description + location + suggested fix
- [ ] Issue 2: Description + location + suggested fix

### 🟡 MAJOR Issues (Should Fix)
[List major issues that should be addressed]
- [ ] Issue 1: Description + location + suggested fix
- [ ] Issue 2: Description + location + suggested fix

### 🔵 MINOR Issues (Nice to Have)
[List minor improvements]
- [ ] Issue 1: Description + location
- [ ] Issue 2: Description + location

### ℹ️ Suggestions
[Optional improvements or alternative approaches]
- Suggestion 1
- Suggestion 2

## Detailed Review

### Architecture & Design
[Comments on overall structure, patterns, and design decisions]

### Code Quality
**Strengths:**
- Strength 1
- Strength 2

**Areas for Improvement:**
- Improvement 1
- Improvement 2

### Testing
- Coverage: [X%] (Minimum 80% required)
- Test quality: [Assessment]
- Missing tests: [List any gaps]

### Standards Compliance
- [ ] run function pattern followed
- [ ] No global commands/init()
- [ ] Dependency injection used
- [ ] Errors properly wrapped
- [ ] Black-box testing
- [ ] Conventional commits
- [ ] golangci-lint passes

## Recommendation
- [ ] ✅ Approve (no blocking issues)
- [ ] 🔄 Request Changes (critical or major issues present)
- [ ] 💬 Comment (questions or suggestions only)

---
[Additional notes or context]
```

---

## Quick Review Template

Use for smaller changes or focused reviews.

```markdown
# Quick Review: [File/Function Name]

## Issues Found

### 🔴 Critical
[List if any]

### 🟡 Major
[List if any]

### 🔵 Minor
[List if any]

## Specific Feedback

**File: [filename]**
Lines [X-Y]:
```go
[problematic code]
```
**Issue:** [Description]
**Fix:** 
```go
[suggested fix]
```

---

## Summary
- Critical issues: [count]
- Major issues: [count]
- Overall: [Approve/Request Changes]
```

---

## Focused Review Templates

### Security Review

```markdown
# Security Review

## Security Checks
- [ ] No SQL injection vulnerabilities
- [ ] Input validation present
- [ ] Secrets not hardcoded
- [ ] Error messages don't leak sensitive info
- [ ] Authentication/authorization properly implemented
- [ ] Rate limiting for public endpoints
- [ ] Context timeouts set appropriately

## Findings
[List any security concerns with severity]

## Recommendation
[Security assessment]
```

### Performance Review

```markdown
# Performance Review

## Performance Checks
- [ ] No N+1 queries
- [ ] Appropriate indexes used
- [ ] Goroutines properly managed
- [ ] Connection pooling configured
- [ ] Caching strategy sound
- [ ] No memory leaks (resources closed)
- [ ] Slice capacity pre-allocated when known

## Findings
[List any performance concerns]

## Recommendations
[Performance improvement suggestions]
```

### Testing Review

```markdown
# Testing Review

## Test Coverage
- Overall: [X%] (≥80% required)
- New code: [Y%]
- Critical paths: [Z%]

## Test Quality Checks
- [ ] Black-box testing (separate package)
- [ ] Table-driven tests used
- [ ] Using testify assertions
- [ ] Tests are deterministic
- [ ] No flaky tests
- [ ] Edge cases covered
- [ ] Error paths tested

## Findings
[List any testing concerns]

## Missing Tests
[Specify what needs test coverage]

## Recommendation
[Pass/Fail based on coverage and quality]
```

### Standards Compliance Review

```markdown
# Standards Compliance Review

## Core Standards
- [ ] main uses run function pattern
- [ ] Dependencies injected via constructors
- [ ] No global commands or init() for flags
- [ ] Interfaces small (1-3 methods) and defined where used
- [ ] Modern Go features used (any, slices/maps, min/max, slog)
- [ ] Errors wrapped with fmt.Errorf("%w", err)
- [ ] Guard clauses used (no nested else)
- [ ] No panic in normal flow

## Naming
- [ ] CamelCase for exported, camelCase for unexported
- [ ] Acronyms uppercase (ServeHTTP, ParseURL)
- [ ] No Get prefix for getters
- [ ] Package names lowercase, single word

## Concurrency
- [ ] APIs synchronous by default
- [ ] Context passed as first argument
- [ ] Goroutine lifecycle clear
- [ ] Mutex for state, channels for signaling

## HTTP (if applicable)
- [ ] Server struct pattern
- [ ] Handlers return closures
- [ ] Go 1.22+ routing used

## Testing
- [ ] Black-box tests (separate package)
- [ ] Table-driven tests
- [ ] ≥80% coverage

## Commits
- [ ] Conventional commits format
- [ ] Types correct (fix/feat/etc)

## Findings
[List any non-compliance]

## Overall Compliance
[Pass/Fail assessment]
```

---

## Example Reviews

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

**🔴 CRITICAL:** Goroutine leak - no way to stop this goroutine

**Fix:** Add context for cancellation:
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

### Example 2: Major Issue - Not Wrapping Errors

```markdown
**File: `internal/service/user.go`**
Lines 23-30:
```go
func (s *Service) GetUser(id string) (*User, error) {
    user, err := s.repo.Get(id)
    if err != nil {
        return nil, err  // Not wrapped
    }
    return user, nil
}
```

**🟡 MAJOR:** Errors not wrapped, losing context

**Fix:** Wrap errors with fmt.Errorf:
```go
func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
    user, err := s.repo.Get(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get user %s: %w", id, err)
    }
    return user, nil
}
```

**Also note:** Missing context parameter (should be first arg)
```

### Example 3: Minor Issue - Slice Capacity

```markdown
**File: `internal/transformer/transform.go`**
Lines 15-20:
```go
func Transform(items []Item) []Result {
    var results []Result
    for _, item := range items {
        results = append(results, transform(item))
    }
    return results
}
```

**🔵 MINOR:** Slice capacity not pre-allocated

**Fix:** Pre-allocate for better performance:
```go
func Transform(items []Item) []Result {
    results := make([]Result, 0, len(items))
    for _, item := range items {
        results = append(results, transform(item))
    }
    return results
}
```
```

---

## Review Workflow

1. **Start with high-level**: Architecture, design, approach
2. **Check critical items first**: Security, goroutine leaks, context issues
3. **Review standards compliance**: Use checklist
4. **Examine test coverage**: Must be ≥80%
5. **Look for anti-patterns**: Reference anti-patterns.md
6. **Provide constructive feedback**: Include code examples
7. **Prioritize issues**: Critical → Major → Minor → Suggestions
8. **Give overall recommendation**: Approve, Request Changes, or Comment
