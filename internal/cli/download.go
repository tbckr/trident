package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/tbckr/trident/internal/appdir"
	providers "github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
)

const detectPatternsURL = "https://raw.githubusercontent.com/tbckr/trident/refs/heads/main/internal/detect/patterns.yaml"

func newDownloadCmd(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "download",
		Short:   "Download trident data files",
		GroupID: "utility",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newDownloadDetectCmd(d))
	return cmd
}

func newDownloadDetectCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "Download latest detect patterns from GitHub",
		Long: `Download the latest provider detection patterns from the trident GitHub repository.

The patterns are saved to ~/.config/trident/detect-downloaded.yaml and are used
by the detect and apex commands as an override over the embedded patterns.

PAP level: AMBER (makes an outbound HTTPS request to GitHub).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !pap.Allows(pap.MustParse(d.cfg.PAPLimit), pap.AMBER) {
				return fmt.Errorf("%w: %q requires PAP %s but limit is %s",
					services.ErrPAPBlocked, "download detect", pap.AMBER, pap.MustParse(d.cfg.PAPLimit))
			}

			client, err := httpclient.New(d.cfg.Proxy, d.cfg.UserAgent, d.logger, d.cfg.Verbose)
			if err != nil {
				return fmt.Errorf("creating HTTP client: %w", err)
			}

			resp, err := client.R().SetContext(cmd.Context()).Get(detectPatternsURL)
			if err != nil {
				return fmt.Errorf("downloading detect patterns: %w", err)
			}
			if resp.Response == nil {
				return fmt.Errorf("downloading detect patterns: transport error (no response)")
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("downloading detect patterns: unexpected status %d", resp.StatusCode)
			}

			var validated providers.Patterns
			if err := yaml.Unmarshal(resp.Bytes(), &validated); err != nil {
				return fmt.Errorf("validating downloaded patterns: %w", err)
			}

			dir, err := appdir.ConfigDir()
			if err != nil {
				return fmt.Errorf("getting config dir: %w", err)
			}
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return fmt.Errorf("creating config dir: %w", err)
			}

			path := filepath.Join(dir, "detect-downloaded.yaml")
			verb := "saved to"
			if _, err := os.Stat(path); err == nil {
				verb = "updated at"
			}

			tmp, err := os.CreateTemp(dir, "detect-downloaded-*.yaml")
			if err != nil {
				return fmt.Errorf("creating temp file: %w", err)
			}
			tmpName := tmp.Name()
			defer func() { _ = os.Remove(tmpName) }()

			if _, err := tmp.Write(resp.Bytes()); err != nil {
				_ = tmp.Close()
				return fmt.Errorf("writing detect patterns: %w", err)
			}
			if err := tmp.Close(); err != nil {
				return fmt.Errorf("closing temp file: %w", err)
			}
			if err := os.Rename(tmpName, path); err != nil { //nolint:gosec // tmpName is created by os.CreateTemp in the same controlled dir
				return fmt.Errorf("installing detect patterns: %w", err)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Detect patterns %s %s\n", verb, path)
			return err
		},
	}
}
