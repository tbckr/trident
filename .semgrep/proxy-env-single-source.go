// Test fixture for the trident-proxy-env-single-source rule, run by semgrep's
// test mode (see the semgrep workflow). Not compiled: Go tooling skips dot
// directories. The annotation comments below name the rule and mark the next
// line as a line it must flag or must leave alone.
package fixtures

import (
	"context"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/http/httpproxy"
	"golang.org/x/net/proxy"
)

func environmentReads() {
	// ruleid: trident-proxy-env-single-source
	_ = os.Getenv("HTTPS_PROXY")
	// ruleid: trident-proxy-env-single-source
	_ = os.Getenv("http_proxy")
	// ruleid: trident-proxy-env-single-source
	_, _ = os.LookupEnv("no_proxy")
	// ruleid: trident-proxy-env-single-source
	_ = os.Getenv("All_Proxy")
	// ruleid: trident-proxy-env-single-source
	_ = os.Getenv("REQUEST_METHOD")
	// ok: trident-proxy-env-single-source
	_ = os.Getenv("HOME")
	// ok: trident-proxy-env-single-source
	_ = os.Getenv("HTTPS_PROXY_NOTE")
}

func environmentReaders(ctx context.Context, u *url.URL) {
	// ruleid: trident-proxy-env-single-source
	_ = http.ProxyFromEnvironment
	// ruleid: trident-proxy-env-single-source
	_ = httpproxy.FromEnvironment()
	// ruleid: trident-proxy-env-single-source
	_ = proxy.FromEnvironment()
	// ruleid: trident-proxy-env-single-source
	_ = proxy.FromEnvironmentUsing(proxy.Direct)
	// ruleid: trident-proxy-env-single-source
	_, _ = proxy.Dial(ctx, "tcp", "example.com:443")
	// ok: trident-proxy-env-single-source
	_, _ = proxy.FromURL(u, proxy.Direct)
}
