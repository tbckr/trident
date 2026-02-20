package config_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/tbckr/trident/internal/config"
)

func TestCompleteOutputFormat(t *testing.T) {
	vals, directive := config.CompleteOutputFormat(nil, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.ElementsMatch(t, []string{"text", "json", "plain"}, vals)
}

func TestCompleteOutputFormat_Prefix(t *testing.T) {
	// prefix is unused by the function; return set must be identical regardless
	vals, directive := config.CompleteOutputFormat(nil, nil, "j")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.ElementsMatch(t, []string{"text", "json", "plain"}, vals)
}

func TestCompletePAPLevel(t *testing.T) {
	vals, directive := config.CompletePAPLevel(nil, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.ElementsMatch(t, []string{"red", "amber", "green", "white"}, vals)
}

func TestCompletePAPLevel_Prefix(t *testing.T) {
	// prefix is unused; full set returned always
	vals, directive := config.CompletePAPLevel(nil, nil, "g")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.ElementsMatch(t, []string{"red", "amber", "green", "white"}, vals)
}
