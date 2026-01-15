# HTTP Service Patterns

Based on Mat Ryer's "How I Write HTTP Web Services after 13 Years".

## Server Structure

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
```

## Routes Definition

```go
func (s *server) routes() {
    s.router.HandleFunc("GET /api/items", s.handleItemsList())
    s.router.HandleFunc("POST /api/items", s.handleItemsCreate())
    s.router.HandleFunc("GET /api/items/{id}", s.handleItemsGet())
    s.router.HandleFunc("PUT /api/items/{id}", s.handleItemsUpdate())
    s.router.HandleFunc("DELETE /api/items/{id}", s.handleItemsDelete())
}
```

## Handler Pattern

Handlers return handler functions (closure pattern):

```go
func (s *server) handleItemsList() http.HandlerFunc {
    // One-time setup, executes once when route is registered
    type response struct {
        Items []Item `json:"items"`
    }
    
    return func(w http.ResponseWriter, r *http.Request) {
        // Request handling, executes on every request
        ctx := r.Context()
        
        items, err := s.getItems(ctx)
        if err != nil {
            s.respond(w, r, nil, http.StatusInternalServerError)
            return
        }
        
        s.respond(w, r, response{Items: items}, http.StatusOK)
    }
}
```

## Middleware Pattern

```go
func (s *server) logging() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            next.ServeHTTP(w, r)
            s.logger.Info("request",
                slog.String("method", r.Method),
                slog.String("path", r.URL.Path),
                slog.Duration("duration", time.Since(start)),
            )
        })
    }
}

func (s *server) routes() {
    s.router.Handle("/api/", s.logging()(s.apiRouter()))
}
```

## Response Helpers

```go
func (s *server) respond(w http.ResponseWriter, r *http.Request, data any, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    
    if data != nil {
        if err := json.NewEncoder(w).Encode(data); err != nil {
            s.logger.Error("failed to encode response",
                slog.String("error", err.Error()),
            )
        }
    }
}

func (s *server) decode(r *http.Request, v any) error {
    return json.NewDecoder(r.Body).Decode(v)
}
```

## Error Response

```go
type errorResponse struct {
    Error string `json:"error"`
}

func (s *server) respondError(w http.ResponseWriter, r *http.Request, err error, status int) {
    s.respond(w, r, errorResponse{Error: err.Error()}, status)
}
```

## Path Parameters (Go 1.22+)

```go
func (s *server) handleItemsGet() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        
        item, err := s.getItem(r.Context(), id)
        if err != nil {
            s.respondError(w, r, err, http.StatusNotFound)
            return
        }
        
        s.respond(w, r, item, http.StatusOK)
    }
}
```

## Testing Servers

```go
func TestHandleItemsList(t *testing.T) {
    srv := newTestServer(t)
    
    req := httptest.NewRequest("GET", "/api/items", nil)
    w := httptest.NewRecorder()
    
    srv.ServeHTTP(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("expected status 200, got %d", w.Code)
    }
}
```
