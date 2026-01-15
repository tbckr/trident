# Testing Patterns

## Black-Box Testing

ALWAYS write tests in a separate test package. This ensures testing only through the public API.

```go
// In file: internal/handler/handler.go
package handler

type Handler struct {
    db Database
}

func NewHandler(db Database) *Handler {
    return &Handler{db: db}
}

func (h *Handler) ProcessItem(ctx context.Context, id string) error {
    // implementation
    return nil
}
```

```go
// In file: internal/handler/handler_test.go
package handler_test  // Note: handler_test, not handler

import (
    "context"
    "testing"
    
    "github.com/user/project/internal/handler"
    "github.com/stretchr/testify/assert"
)

func TestHandler_ProcessItem(t *testing.T) {
    // Can only access exported types and functions
    h := handler.NewHandler(mockDB)
    
    err := h.ProcessItem(context.Background(), "123")
    assert.NoError(t, err)
}
```

## Table-Driven Tests

The standard pattern for Go tests:

```go
package calculator_test

import (
    "testing"
    
    "github.com/user/project/calculator"
    "github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a        int
        b        int
        expected int
    }{
        {
            name:     "positive numbers",
            a:        2,
            b:        3,
            expected: 5,
        },
        {
            name:     "negative numbers",
            a:        -2,
            b:        -3,
            expected: -5,
        },
        {
            name:     "mixed signs",
            a:        10,
            b:        -5,
            expected: 5,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := calculator.Add(tt.a, tt.b)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## Testing with Testify

### Assertions

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
    // assert continues on failure
    assert.Equal(t, expected, actual, "values should match")
    assert.NoError(t, err)
    assert.True(t, condition)
    assert.NotNil(t, obj)
    
    // require stops test on failure (use for setup that must succeed)
    require.NoError(t, err, "setup must not fail")
    require.NotNil(t, db, "database connection required")
}
```

### Test Suites

```go
package service_test

import (
    "testing"
    
    "github.com/stretchr/testify/suite"
    "github.com/user/project/service"
)

type ServiceTestSuite struct {
    suite.Suite
    svc *service.Service
}

func (s *ServiceTestSuite) SetupTest() {
    // Runs before each test
    s.svc = service.New()
}

func (s *ServiceTestSuite) TearDownTest() {
    // Runs after each test
    s.svc.Close()
}

func (s *ServiceTestSuite) TestServiceMethod() {
    result, err := s.svc.DoSomething()
    s.NoError(err)
    s.Equal("expected", result)
}

func TestServiceTestSuite(t *testing.T) {
    suite.Run(t, new(ServiceTestSuite))
}
```

## Mock Interfaces

Define interfaces where they're used, implement mocks for testing:

```go
// In file: internal/service/service.go
package service

type Repository interface {
    Get(ctx context.Context, id string) (*Item, error)
    Save(ctx context.Context, item *Item) error
}

type Service struct {
    repo Repository
}

func New(repo Repository) *Service {
    return &Service{repo: repo}
}
```

```go
// In file: internal/service/service_test.go
package service_test

import (
    "context"
    "errors"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/user/project/internal/service"
)

type mockRepository struct {
    mock.Mock
}

func (m *mockRepository) Get(ctx context.Context, id string) (*service.Item, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*service.Item), args.Error(1)
}

func (m *mockRepository) Save(ctx context.Context, item *service.Item) error {
    args := m.Called(ctx, item)
    return args.Error(0)
}

func TestService_Get(t *testing.T) {
    repo := new(mockRepository)
    svc := service.New(repo)
    
    expectedItem := &service.Item{ID: "123", Name: "Test"}
    repo.On("Get", mock.Anything, "123").Return(expectedItem, nil)
    
    item, err := svc.Get(context.Background(), "123")
    
    assert.NoError(t, err)
    assert.Equal(t, expectedItem, item)
    repo.AssertExpectations(t)
}
```

## Test Helpers

```go
package service_test

import (
    "testing"
    "github.com/user/project/internal/service"
)

func newTestService(t *testing.T) *service.Service {
    t.Helper()
    
    // Setup common test dependencies
    repo := newMockRepository()
    return service.New(repo)
}

func assertItemEqual(t *testing.T, expected, actual *service.Item) {
    t.Helper()
    
    if expected.ID != actual.ID {
        t.Errorf("ID mismatch: expected %s, got %s", expected.ID, actual.ID)
    }
    if expected.Name != actual.Name {
        t.Errorf("Name mismatch: expected %s, got %s", expected.Name, actual.Name)
    }
}
```

## Coverage Requirements

Minimum 80% coverage is mandatory. Check with:

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Integration with CI:

```yaml
# .github/workflows/test.yml
- name: Test with coverage
  run: go test -race -coverprofile=coverage.out -covermode=atomic ./...
  
- name: Check coverage threshold
  run: |
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$coverage < 80" | bc -l) )); then
      echo "Coverage $coverage% is below 80%"
      exit 1
    fi
```
