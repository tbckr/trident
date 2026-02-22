package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResolver_EmptyProxy(t *testing.T) {
	r, err := NewResolver("")
	require.NoError(t, err)
	assert.NotNil(t, r)
	assert.Nil(t, r.Dial, "standard resolver should have nil Dial")
}

func TestNewResolver_NonSocks5Proxy(t *testing.T) {
	for _, u := range []string{"http://proxy.example.com:8080", "https://proxy.example.com:8080"} {
		r, err := NewResolver(u)
		require.NoError(t, err, "proxy=%s", u)
		assert.NotNil(t, r)
		assert.Nil(t, r.Dial, "non-socks5 proxy should use standard resolver")
	}
}

func TestNewResolver_Socks5Proxy(t *testing.T) {
	r, err := NewResolver("socks5://127.0.0.1:1080")
	require.NoError(t, err)
	assert.NotNil(t, r)
	assert.NotNil(t, r.Dial, "socks5 proxy should set a custom Dial function")
	assert.True(t, r.PreferGo, "socks5 resolver must use Go resolver (PreferGo=true)")
}

func TestNewResolver_AllProxy_Socks5(t *testing.T) {
	t.Setenv("ALL_PROXY", "socks5://127.0.0.1:1080")
	r, err := NewResolver("")
	require.NoError(t, err)
	assert.NotNil(t, r)
	assert.NotNil(t, r.Dial, "ALL_PROXY socks5 should set a custom Dial function")
	assert.True(t, r.PreferGo, "ALL_PROXY socks5 resolver must use Go resolver (PreferGo=true)")
}

func TestNewResolver_AllProxy_Http(t *testing.T) {
	t.Setenv("ALL_PROXY", "http://proxy.example.com:8080")
	r, err := NewResolver("")
	require.NoError(t, err)
	assert.NotNil(t, r)
	assert.Nil(t, r.Dial, "HTTP ALL_PROXY should fall back to standard resolver (nil Dial)")
}
