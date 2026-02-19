package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTerminalWidth_NonTerminal(t *testing.T) {
	var buf bytes.Buffer
	assert.Equal(t, defaultTermWidth, TerminalWidth(&buf))
}
