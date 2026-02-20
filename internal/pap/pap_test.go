package pap_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/pap"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		expected pap.Level
	}{
		{"red", pap.RED},
		{"RED", pap.RED},
		{"Red", pap.RED},
		{"amber", pap.AMBER},
		{"AMBER", pap.AMBER},
		{"green", pap.GREEN},
		{"GREEN", pap.GREEN},
		{"white", pap.WHITE},
		{"WHITE", pap.WHITE},
		{"  amber  ", pap.AMBER},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := pap.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParse_Invalid(t *testing.T) {
	for _, bad := range []string{"", "blue", "grey", "0", "1"} {
		_, err := pap.Parse(bad)
		require.Error(t, err, "expected error for %q", bad)
		assert.Contains(t, err.Error(), "unknown PAP level")
	}
}

func TestString(t *testing.T) {
	assert.Equal(t, "red", pap.RED.String())
	assert.Equal(t, "amber", pap.AMBER.String())
	assert.Equal(t, "green", pap.GREEN.String())
	assert.Equal(t, "white", pap.WHITE.String())
}

func TestAllows(t *testing.T) {
	tests := []struct {
		limit   pap.Level
		service pap.Level
		allowed bool
	}{
		// RED limit: most restrictive — only non-detectable (offline/local) services allowed
		{pap.RED, pap.RED, true},
		{pap.RED, pap.AMBER, false},
		{pap.RED, pap.GREEN, false},
		{pap.RED, pap.WHITE, false},
		// AMBER limit: allow 3rd-party APIs and below; block direct-interaction services
		{pap.AMBER, pap.RED, true},
		{pap.AMBER, pap.AMBER, true},
		{pap.AMBER, pap.GREEN, false},
		{pap.AMBER, pap.WHITE, false},
		// GREEN limit: allow direct target interaction and below; block unrestricted
		{pap.GREEN, pap.RED, true},
		{pap.GREEN, pap.AMBER, true},
		{pap.GREEN, pap.GREEN, true},
		{pap.GREEN, pap.WHITE, false},
		// WHITE limit: unrestricted — all services allowed
		{pap.WHITE, pap.RED, true},
		{pap.WHITE, pap.AMBER, true},
		{pap.WHITE, pap.GREEN, true},
		{pap.WHITE, pap.WHITE, true},
	}
	for _, tt := range tests {
		t.Run(tt.limit.String()+"/"+tt.service.String(), func(t *testing.T) {
			assert.Equal(t, tt.allowed, pap.Allows(tt.limit, tt.service))
		})
	}
}

func TestMustParse(t *testing.T) {
	tests := []struct {
		input    string
		expected pap.Level
	}{
		{"red", pap.RED},
		{"amber", pap.AMBER},
		{"green", pap.GREEN},
		{"white", pap.WHITE},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, pap.MustParse(tt.input))
		})
	}
}

func TestMustParse_Panics(t *testing.T) {
	require.Panics(t, func() { pap.MustParse("invalid") })
}

func TestLevelOrdering(t *testing.T) {
	// Verify the numeric ordering: RED < AMBER < GREEN < WHITE
	// (ascending activity level — higher = more active/invasive)
	assert.Less(t, int(pap.RED), int(pap.AMBER))
	assert.Less(t, int(pap.AMBER), int(pap.GREEN))
	assert.Less(t, int(pap.GREEN), int(pap.WHITE))
}
