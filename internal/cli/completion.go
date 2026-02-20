package cli

import "github.com/spf13/cobra"

func newCompletionCmd() *cobra.Command {
	completion := &cobra.Command{
		Use:     "completion [bash|zsh|fish|powershell]",
		Short:   "Generate shell completion scripts",
		GroupID: "utility",
		Long: `Generate shell completion scripts for trident.

To load completions:

Bash:
  $ source <(trident completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ trident completion bash > /etc/bash_completion.d/trident
  # macOS:
  $ trident completion bash > $(brew --prefix)/etc/bash_completion.d/trident

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it first:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  $ source <(trident completion zsh)

  # To load completions for each session, execute once:
  $ trident completion zsh > "${fpath[1]}/_trident"

Fish:
  $ trident completion fish | source

  # To load completions for each session, execute once:
  $ trident completion fish > ~/.config/fish/completions/trident.fish

PowerShell:
  PS> trident completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, add the output of the above
  # command to your PowerShell profile.`,
		// Override root's PersistentPreRunE — buildDeps must not run during
		// tab-completion because it has filesystem side effects (creates config
		// dir and file). This is the only subcommand permitted to override
		// PersistentPreRunE without calling buildDeps.
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}

	completion.AddCommand(
		newCompletionBashCmd(),
		newCompletionZshCmd(),
		newCompletionFishCmd(),
		newCompletionPowerShellCmd(),
	)

	return completion
}

func newCompletionBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "bash",
		Short:                 "Generate bash completion script",
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		Long: `Generate the autocompletion script for bash.

This script depends on the 'bash-completion' package. If not installed, you can
install it via your OS package manager.

To load completions in your current shell session:
  $ source <(trident completion bash)

To load completions for every new session, execute once:
  # Linux:
  $ trident completion bash > /etc/bash_completion.d/trident
  # macOS:
  $ trident completion bash > $(brew --prefix)/etc/bash_completion.d/trident

You will need to start a new shell for the setup to take effect.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenBashCompletionV2(cmd.OutOrStdout(), true)
		},
	}
}

func newCompletionZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "zsh",
		Short:                 "Generate zsh completion script",
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		Long: `Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment, enable it once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:
  $ source <(trident completion zsh)

To load completions for every new session, execute once:
  $ trident completion zsh > "${fpath[1]}/_trident"

You will need to start a new shell for the setup to take effect.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		},
	}
}

func newCompletionFishCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "fish",
		Short:                 "Generate fish completion script",
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		Long: `Generate the autocompletion script for the fish shell.

To load completions in your current shell session:
  $ trident completion fish | source

To load completions for every new session, execute once:
  $ trident completion fish > ~/.config/fish/completions/trident.fish

You will need to start a new shell for the setup to take effect.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
		},
	}
}

func newCompletionPowerShellCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "powershell",
		Short:                 "Generate PowerShell completion script",
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		Long: `Generate the autocompletion script for PowerShell.

To load completions in your current shell session:
  PS> trident completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your PowerShell profile.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		},
	}
}
