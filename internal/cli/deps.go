package cli

import (
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/imroc/req/v3"
	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/config"
	providers "github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/resolver"
)

// deps holds fully-resolved runtime dependencies for a subcommand.
type deps struct {
	logger   *slog.Logger
	cfg      *config.Config
	doDefang bool
	papLevel pap.Level
}

// buildDeps resolves config, logger, output format, PAP level, and defang flag.
func buildDeps(cmd *cobra.Command, stderr io.Writer) (*deps, error) {
	cfg, err := config.Load(cmd.Flags())
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if cfg.Defang && cfg.NoDefang {
		return nil, fmt.Errorf("--defang and --no-defang are mutually exclusive")
	}

	if cfg.Concurrency < 1 {
		return nil, fmt.Errorf("--concurrency must be at least 1, got %d", cfg.Concurrency)
	}

	level := slog.LevelInfo
	if cfg.Verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))

	format := output.Format(cfg.Output)
	switch format {
	case output.FormatTable, output.FormatJSON, output.FormatText:
	default:
		return nil, fmt.Errorf("invalid output format %q: must be \"table\", \"json\", or \"text\"", cfg.Output)
	}

	papLevel, err := pap.Parse(cfg.PAPLimit)
	if err != nil {
		return nil, fmt.Errorf("invalid PAP limit %q: %w", cfg.PAPLimit, err)
	}

	doDefang := output.ResolveDefang(papLevel, format, cfg.Defang, cfg.NoDefang)

	return &deps{cfg: cfg, logger: logger, doDefang: doDefang, papLevel: papLevel}, nil
}

// newHTTPClient creates a new HTTP client configured with the proxy, user-agent,
// logger, and verbosity from the resolved config.
func (d *deps) newHTTPClient() (*req.Client, error) {
	client, err := httpclient.New(d.cfg.Proxy, d.cfg.UserAgent, d.logger, d.cfg.Verbose)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP client: %w", err)
	}
	return client, nil
}

// newResolver creates a new DNS resolver configured with the proxy from the
// resolved config.
func (d *deps) newResolver() (*net.Resolver, error) {
	r, err := resolver.NewResolver(d.cfg.Proxy)
	if err != nil {
		return nil, fmt.Errorf("creating DNS resolver: %w", err)
	}
	return r, nil
}

// loadPatterns loads the provider detection patterns, prepending any
// user-supplied override file from config.
func (d *deps) loadPatterns() (providers.Patterns, error) {
	paths, err := providers.DefaultPatternPaths()
	if err != nil {
		return providers.Patterns{}, fmt.Errorf("resolving pattern paths: %w", err)
	}
	if d.cfg.DetectPatterns.File != "" {
		paths = append([]string{d.cfg.DetectPatterns.File}, paths...)
	}
	patterns, err := providers.LoadPatterns(paths...)
	if err != nil {
		return providers.Patterns{}, fmt.Errorf("loading detect patterns: %w", err)
	}
	return patterns, nil
}

// writeResult formats and writes a service result to stdout.
// When d.doDefang is true the writer is wrapped with DefangWriter.
func writeResult(stdout io.Writer, d *deps, result any) error {
	w := stdout
	if d.doDefang {
		w = &output.DefangWriter{Inner: stdout}
	}
	if err := output.Write(w, output.Format(d.cfg.Output), result); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}
