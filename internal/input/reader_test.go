package input_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/input"
)

func TestRead_Basic(t *testing.T) {
	r := strings.NewReader("example.com\ngoogle.com\n")
	inputs, err := input.Read(r)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com", "google.com"}, inputs)
}

func TestRead_TrimsWhitespace(t *testing.T) {
	r := strings.NewReader("  example.com  \n\tgoogle.com\t\n")
	inputs, err := input.Read(r)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com", "google.com"}, inputs)
}

func TestRead_DropsEmptyLines(t *testing.T) {
	r := strings.NewReader("example.com\n\n\ngoogle.com\n")
	inputs, err := input.Read(r)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com", "google.com"}, inputs)
}

func TestRead_Empty(t *testing.T) {
	r := strings.NewReader("")
	inputs, err := input.Read(r)
	require.NoError(t, err)
	assert.Nil(t, inputs)
}

func TestRead_WhitespaceOnly(t *testing.T) {
	r := strings.NewReader("   \n\t\n  \n")
	inputs, err := input.Read(r)
	require.NoError(t, err)
	assert.Nil(t, inputs)
}

func TestRead_NoTrailingNewline(t *testing.T) {
	r := strings.NewReader("example.com")
	inputs, err := input.Read(r)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com"}, inputs)
}
