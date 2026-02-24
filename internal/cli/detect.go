package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	providers "github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/resolver"
	detectsvc "github.com/tbckr/trident/internal/services/detect"
)

func newDetectCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "detect [domain...]",
		Short:   "Detect CDN, email, and DNS hosting providers",
		GroupID: "services",
		Long: `Detect CDN, email, and DNS hosting providers for one or more domains.

Queries CNAME (apex and www), MX, and NS records and matches them against
known provider patterns to identify cloud services in use.

PAP level: GREEN (direct interaction with the target's DNS servers).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  trident detect example.com`,
		Args:    cobra.ArbitraryArgs,
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolver.NewResolver(d.cfg.Proxy)
			if err != nil {
				return fmt.Errorf("creating DNS resolver: %w", err)
			}
			paths, err := providers.DefaultPatternPaths()
			if err != nil {
				return fmt.Errorf("resolving pattern paths: %w", err)
			}
			patterns, err := providers.LoadPatterns(paths...)
			if err != nil {
				return fmt.Errorf("loading detect patterns: %w", err)
			}
			svc := detectsvc.NewService(r, d.logger, patterns)
			return runServiceCmd(cmd, d, svc, args)
		},
	}
}
