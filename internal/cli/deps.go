package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/imroc/req/v3"
	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/config"
	providers "github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/doh"
	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/proxyconf"
	"github.com/tbckr/trident/internal/resolver"
)

// deps holds fully-resolved runtime dependencies for a subcommand.
type deps struct {
	logger *slog.Logger
	cfg    *config.Config
	// proxy is the run's one proxy resolution (cfg.Proxy and the proxy
	// environment). A deferred error in it aborts only the commands that
	// build a network client.
	proxy    *proxyconf.Settings
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

	if warn := config.WarnInsecurePermissions(cfg.ConfigFile); warn != "" {
		logger.Warn(warn)
	}

	// Resolve never fails: a proxy problem surfaces only when a command
	// builds a network client.
	proxy := proxyconf.Resolve(cfg.Proxy, os.Getenv)

	return &deps{cfg: cfg, logger: logger, proxy: proxy, doDefang: doDefang, papLevel: papLevel}, nil
}

// newHTTPClient creates a new HTTP client configured with the proxy settings,
// user-agent, logger, and verbosity from the resolved config. It fails with
// the run's proxy error, if any.
func (d *deps) newHTTPClient() (*req.Client, error) {
	client, err := httpclient.New(d.proxy, d.cfg.UserAgent, d.logger, d.cfg.Verbose)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP client: %w", err)
	}
	return client, nil
}

// newDoHClient creates the HTTP client for DNS-over-HTTPS requests to
// doh.Endpoint, configured like newHTTPClient. proxied reports whether a proxy
// applies to the endpoint. It runs inside the commands that need it, not in
// buildDeps: it fails with the run's proxy error (such as
// HTTPS_PROXY=socks4://), which must not break the commands that do not use
// the network.
func (d *deps) newDoHClient() (client *req.Client, proxied bool, err error) {
	client, proxied, err = httpclient.NewDoH(d.proxy, doh.Endpoint, d.cfg.UserAgent, d.logger, d.cfg.Verbose)
	if err != nil {
		return nil, false, fmt.Errorf("creating DoH client: %w", err)
	}
	return client, proxied, nil
}

// dnsUnproxiedWarning is logged by newResolver when DNS cannot use the
// configured proxy.
const dnsUnproxiedWarning = "DNS lookups cannot go through the configured proxy and use the system resolver; " +
	"use a socks5h:// proxy (apex already resolves through the proxy via DoH)"

// newResolver creates the DNS resolver for the run's proxy settings. It fails
// with the run's proxy error, if any. Behind an HTTP(S) proxy, which cannot
// carry DNS, the lookups use the system resolver: newResolver warns about it,
// and it is the only place that does, so only the commands that resolve
// through it show the warning.
func (d *deps) newResolver() (*resolver.Resolver, error) {
	r, err := resolver.NewResolver(d.proxy)
	if err != nil {
		return nil, fmt.Errorf("creating DNS resolver: %w", err)
	}
	if d.proxy.DNSUnproxied() {
		d.logger.Warn(dnsUnproxiedWarning, "proxy", proxyconf.Redact(d.proxy.HTTPSProxy()))
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
