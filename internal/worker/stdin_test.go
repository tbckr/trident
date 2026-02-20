package worker_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/worker"
)

func TestReadInputs_Basic(t *testing.T) {
	r := strings.NewReader("example.com\ngoogle.com\n")
	inputs, err := worker.ReadInputs(r)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com", "google.com"}, inputs)
}

func TestReadInputs_TrimsWhitespace(t *testing.T) {
	r := strings.NewReader("  example.com  \n\tgoogle.com\t\n")
	inputs, err := worker.ReadInputs(r)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com", "google.com"}, inputs)
}

func TestReadInputs_DropsEmptyLines(t *testing.T) {
	r := strings.NewReader("example.com\n\n\ngoogle.com\n")
	inputs, err := worker.ReadInputs(r)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com", "google.com"}, inputs)
}

func TestReadInputs_Empty(t *testing.T) {
	r := strings.NewReader("")
	inputs, err := worker.ReadInputs(r)
	require.NoError(t, err)
	assert.Nil(t, inputs)
}

func TestReadInputs_WhitespaceOnly(t *testing.T) {
	r := strings.NewReader("   \n\t\n  \n")
	inputs, err := worker.ReadInputs(r)
	require.NoError(t, err)
	assert.Nil(t, inputs)
}

func TestReadInputs_NoTrailingNewline(t *testing.T) {
	r := strings.NewReader("example.com")
	inputs, err := worker.ReadInputs(r)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com"}, inputs)
}
