// Test fixture for the trident-no-implicit-proxy-env and
// trident-no-req-default-client rules, run by `semgrep --test .semgrep/`. Not
// compiled: Go tooling skips dot directories. `// ruleid:` marks the line after
// it as an expected finding, `// ok:` as one that must not match.
package fixtures

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/imroc/req/v3"
)

func implicitClients(r *http.Request, u string, v url.Values) {
	// ruleid: trident-no-implicit-proxy-env
	_, _ = http.DefaultClient.Do(r)
	// ruleid: trident-no-implicit-proxy-env
	_, _ = http.Get(u)
	// ruleid: trident-no-implicit-proxy-env
	_, _ = http.Head(u)
	// ruleid: trident-no-implicit-proxy-env
	_, _ = http.Post(u, "text/plain", strings.NewReader("x"))
	// ruleid: trident-no-implicit-proxy-env
	_, _ = http.PostForm(u, v)
	// ruleid: trident-no-implicit-proxy-env
	_ = new(http.Client)
	// ruleid: trident-no-implicit-proxy-env
	_ = &http.Client{Timeout: time.Second}
	// ruleid: trident-no-implicit-proxy-env
	_ = http.Client{}
	// ruleid: trident-no-implicit-proxy-env
	_ = &http.Client{Transport: http.DefaultTransport}
	// ruleid: trident-no-implicit-proxy-env
	_, _ = http.DefaultTransport.RoundTrip(r)
	// ruleid: trident-no-implicit-proxy-env
	_ = &http.Client{Transport: http.DefaultTransport.(*http.Transport).Clone()}
}

func cloneWithoutProxy() *http.Transport {
	// ruleid: trident-no-implicit-proxy-env
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ForceAttemptHTTP2 = true
	return tr
}

func explicitClients(tr *http.Transport) {
	// ok: trident-no-implicit-proxy-env
	_ = &http.Client{Transport: tr}
	// ok: trident-no-implicit-proxy-env
	_ = &http.Client{Transport: tr, Timeout: time.Second}
	// ok: trident-no-implicit-proxy-env
	_ = &http.Transport{}
}

// The shape of newDoH in internal/httpclient/doh.go: the clone's Proxy is
// replaced right away.
func cloneWithProxy(p func(*http.Request) (*url.URL, error)) *http.Transport {
	// ok: trident-no-implicit-proxy-env
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = p
	tr.ForceAttemptHTTP2 = true
	return tr
}

func reqDefaults(u string) {
	// ruleid: trident-no-req-default-client
	_ = req.C()
	// ruleid: trident-no-req-default-client
	_ = req.NewClient()
	// ruleid: trident-no-req-default-client
	_ = req.R()
	// ruleid: trident-no-req-default-client
	_ = req.DefaultClient()
	// ruleid: trident-no-req-default-client
	_, _ = req.Get(u)
	// ruleid: trident-no-req-default-client
	_ = req.MustPost(u)
}

func reqFromDeps(c *req.Client, u string) {
	// ok: trident-no-req-default-client
	_, _ = c.R().Get(u)
}
