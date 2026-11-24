package httpclient

import (
	"testing"
	"time"
)

// Exported for the external test package.
var (
	CheckDirectDial       = checkDirectDial
	RefuseLocalDNSProxies = refuseLocalDNSProxies
	NewDoHWithProxyFunc   = newDoH
)

// SetDoHTimeouts sets the DoH request timeout and HTTP/2 ping interval and
// timeout for clients built during the test, and restores them afterwards.
func SetDoHTimeouts(t testing.TB, request, pingInterval, pingTimeout time.Duration) {
	t.Helper()
	oldRequest, oldInterval, oldTimeout := dohRequestTimeout, dohPingInterval, dohPingTimeout
	t.Cleanup(func() {
		dohRequestTimeout, dohPingInterval, dohPingTimeout = oldRequest, oldInterval, oldTimeout
	})
	dohRequestTimeout, dohPingInterval, dohPingTimeout = request, pingInterval, pingTimeout
}
