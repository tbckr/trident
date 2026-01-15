# CLI Patterns with Cobra

## Root Command Constructor

NEVER use package-level variables or init() functions. Always use constructors.

```go
func NewRootCmd(logger *slog.Logger, levelVar *slog.LevelVar) *cobra.Command {
    var configFile string
    
    cmd := &cobra.Command{
        Use:   "myapp",
        Short: "A brief description of your application",
        Long:  `A longer description...`,
        PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
            // Setup that runs before all commands
            if configFile != "" {
                // Load config
            }
            return nil
        },
    }
    
    // Persistent flags available to all subcommands
    cmd.PersistentFlags().StringVar(&configFile, "config", "", "config file")
    cmd.PersistentFlags().StringVar(&levelVar.Level().String(), "log-level", "info", "log level (debug|info|warn|error)")
    
    // Add subcommands
    cmd.AddCommand(NewServeCmd(logger))
    cmd.AddCommand(NewVersionCmd())
    
    return cmd
}
```

## Subcommand Pattern

```go
func NewServeCmd(logger *slog.Logger) *cobra.Command {
    var (
        port int
        host string
    )
    
    cmd := &cobra.Command{
        Use:   "serve",
        Short: "Start the HTTP server",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()
            
            addr := fmt.Sprintf("%s:%d", host, port)
            logger.Info("starting server", slog.String("addr", addr))
            
            // Your server logic here
            return runServer(ctx, addr, logger)
        },
    }
    
    cmd.Flags().IntVar(&port, "port", 8080, "port to listen on")
    cmd.Flags().StringVar(&host, "host", "localhost", "host to bind to")
    
    return cmd
}
```

## Main Function Integration

```go
func main() {
    ctx := context.Background()
    
    // Setup structured logging
    levelVar := &slog.LevelVar{}
    levelVar.Set(slog.LevelInfo)
    
    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: levelVar,
    }))
    
    if err := run(ctx, os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr, logger, levelVar); err != nil {
        logger.Error("application error", slog.String("error", err.Error()))
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
    rootCmd := NewRootCmd(logger, levelVar)
    rootCmd.SetArgs(args)
    rootCmd.SetIn(stdin)
    rootCmd.SetOut(stdout)
    rootCmd.SetErr(stderr)
    
    return rootCmd.ExecuteContext(ctx)
}
```

## Testing CLI Commands

```go
package main_test

import (
    "bytes"
    "context"
    "log/slog"
    "strings"
    "testing"
)

func TestRootCommand(t *testing.T) {
    tests := []struct {
        name    string
        args    []string
        wantErr bool
    }{
        {
            name:    "version command",
            args:    []string{"version"},
            wantErr: false,
        },
        {
            name:    "invalid command",
            args:    []string{"invalid"},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            var stdout, stderr bytes.Buffer
            
            levelVar := &slog.LevelVar{}
            logger := slog.New(slog.NewTextHandler(&stderr, &slog.HandlerOptions{
                Level: levelVar,
            }))
            
            getenv := func(key string) string {
                return ""
            }
            
            err := run(ctx, tt.args, getenv, strings.NewReader(""), &stdout, &stderr, logger, levelVar)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Cobra Testing Best Practices

Use table-driven tests with subtests. Test command parsing, flag validation, and execution separately when possible.

```go
func TestServeCommand(t *testing.T) {
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    
    cmd := NewServeCmd(logger)
    
    // Test flag parsing
    cmd.SetArgs([]string{"--port", "9090", "--host", "0.0.0.0"})
    
    if err := cmd.Execute(); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```
