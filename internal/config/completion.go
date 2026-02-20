package config

import "github.com/spf13/cobra"

// CompleteOutputFormat provides shell completion candidates for the --output flag.
func CompleteOutputFormat(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{"text", "json", "plain"}, cobra.ShellCompDirectiveNoFileComp
}

// CompletePAPLevel provides shell completion candidates for the --pap flag.
func CompletePAPLevel(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{"red", "amber", "green", "white"}, cobra.ShellCompDirectiveNoFileComp
}
