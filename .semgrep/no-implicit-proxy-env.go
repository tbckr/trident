// Test fixture for the trident-no-implicit-proxy-env and
// trident-no-req-default-client rules, run by semgrep's
// test mode (see the semgrep workflow). Not compiled: Go tooling skips dot
// directories. The annotation comments below name the rule and mark the next
// line as a line it must flag or must leave alone.
package fixtures

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/imroc/req/v3"
)

func implicitClients(r *http.Request, u string, v url.Values, c *http.Client) {
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
	// ruleid: trident-no-implicit-proxy-env
	_ = &http.Client{Transport: nil}
	// ruleid: trident-no-implicit-proxy-env
	c.Transport = http.DefaultTransport
	// ruleid: trident-no-implicit-proxy-env
	rt := http.DefaultTransport
	// ruleid: trident-no-implicit-proxy-env
	wrap(http.DefaultTransport)
	_ = rt
}

func wrap(http.RoundTripper) {}

func zeroValueClient() {
	// ruleid: trident-no-implicit-proxy-env
	var c http.Client
	_ = c
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
	// ruleid: trident-no-req-default-client
	_, _ = req.SetContext(context.Background()).Get(u)
	// ruleid: trident-no-req-default-client
	_ = req.T()
}

func reqFromDeps(c *req.Client, u string) {
	// ok: trident-no-req-default-client
	_, _ = c.R().Get(u)
}
