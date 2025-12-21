package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

type RootOptions struct {
	Config      string
	Verbose     bool
	Output      string
	Proxy       string
	UserAgent   string
	PAPLimit    string
	Defang      bool
	NoDefang    bool
	Concurrency int
}

func NewRootCmd(logger *slog.Logger, levelVar *slog.LevelVar, getenv func(string) string) *cobra.Command {
	opts := &RootOptions{}

	cmd := &cobra.Command{
		Use:   "trident",
		Short: "🔥 A high-performance multi-tool for OSINT gathering and security investigations",
		Long: `Trident 🔱 is a statically compiled CLI tool designed for speed and operational security (OpSec).
It automates querying various threat intelligence, network, and identity platforms while 
protecting the investigator's identity via PAP levels, proxies, and rotation.

Trident supports bulk processing via stdin and flexible output formats (JSON, Tables, Plain text).`,
		Example: `  # Bulk DNS lookup from file
  cat domains.txt | trident dns --output json > results.json

  # ASN lookup via SOCKS5 proxy (Tor)
  trident asn 8.8.8.8 --proxy socks5://127.0.0.1:9050

  # Search subdomains on crt.sh with AMBER PAP safety
  trident crtsh example.com --pap-limit amber`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.Verbose {
				levelVar.Set(slog.LevelDebug)
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&opts.Config, "config", "", "config file (default is $HOME/.config/trident/config.yaml)")
	cmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "verbose output")
	cmd.PersistentFlags().StringVarP(&opts.Output, "output", "o", "text", "output format (text|json|plain)")
	cmd.PersistentFlags().StringVar(&opts.Proxy, "proxy", "", "proxy URL")
	cmd.PersistentFlags().StringVar(&opts.UserAgent, "user-agent", "", "custom User-Agent string")
	cmd.PersistentFlags().StringVar(&opts.PAPLimit, "pap-limit", "white", "PAP level limit (white|green|amber|red)")
	cmd.PersistentFlags().BoolVar(&opts.Defang, "defang", false, "force defanging of output")
	cmd.PersistentFlags().BoolVar(&opts.NoDefang, "no-defang", false, "disable defanging of output")
	cmd.PersistentFlags().IntVarP(&opts.Concurrency, "concurrency", "c", 10, "maximum concurrent operations")

	return cmd
}
