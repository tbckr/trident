package proxyconf

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/tbckr/trident/internal/apperr"
)

// SOCKSNameserver is the resolver DNS queries go to, over TCP through the
// proxy, when the DNS proxy is a SOCKS5 proxy. It is a fixed public resolver
// (Quad9, unfiltered) rather than the one from /etc/resolv.conf, which a remote
// proxy cannot reach (it is the proxy's own loopback) and Tor refuses.
const SOCKSNameserver = "9.9.9.10:53"

// settingSource names trident's own setting (--proxy, the config file's proxy
// key, TRIDENT_PROXY) in errors and descriptions.
const settingSource = "proxy"

// route is the proxy one path (https or http) uses and where it came from.
type route struct {
	url    *url.URL // validated; nil means the path goes direct
	source string   // settingSource or the environment variable's exact name
}

// Settings is the one proxy resolution of a run: which proxy https, http and
// DNS use, the NO_PROXY exemptions, and the deferred error if the setting or
// the environment cannot be honoured. It is immutable once built and safe for
// concurrent use.
type Settings struct {
	// err is the one deferred error of the run. While it is set no proxy is
	// handed out and ProxyFunc fails.
	err error

	// viaSetting is true when trident's own setting applies (the environment
	// is then ignored, NO_PROXY included), valid or not.
	viaSetting bool
	// disabled is true for NO_PROXY=*: everything goes direct.
	disabled bool

	https route
	http  route
	// httpsSelected and httpSelected record that a value was selected for the
	// path, valid or not, so that Configured errs toward "proxied" even when
	// the value could not be used.
	httpsSelected bool
	httpSelected  bool

	noProxy      noProxy
	noProxyValue string // the NO_PROXY value in effect, for Describe
	noProxyErr   error  // a NO_PROXY conflict or parse error, for Describe
}

// pair is the value of one lowercase/uppercase variable pair.
type pair struct {
	value  string
	source string // the variable that supplied value; lowercase when both match
	set    bool   // at least one of the two is non-empty
	err    error  // both are set and differ
}

// pick reads a variable pair. An empty variable counts as unset. If both are
// set and differ (byte comparison) the pair has an error and no value. With
// dropUpper the uppercase variable is treated as unset (the CGI rule).
func pick(getenv func(string) string, lower, upper string, dropUpper bool) pair {
	l := getenv(lower)
	u := ""
	if !dropUpper {
		u = getenv(upper)
	}
	switch {
	case l == "" && u == "":
		return pair{}
	case u == "" || l == u:
		return pair{value: l, source: lower, set: true}
	case l == "":
		return pair{value: u, source: upper, set: true}
	default:
		return pair{set: true, err: fmt.Errorf("%w: %s and %s are both set and differ; unset one", apperr.ErrProxyConfig, lower, upper)}
	}
}

// httpOnlyError is the error for a run where only the http variable is set.
// v is that variable's name; the hints name the other variables in its case.
// It never contains the value.
func httpOnlyError(v string) error {
	httpsVar, allVar := "https_proxy", "all_proxy"
	if v == "HTTP_PROXY" {
		httpsVar, allVar = "HTTPS_PROXY", "ALL_PROXY"
	}
	return fmt.Errorf("%w: %s applies only to http:// URLs, but trident's services use https:// and would bypass the proxy; "+
		"set %s (or %s) to the same value, use --proxy socks5h://…, or set NO_PROXY='*' to connect directly "+
		"(dns, detect and cymru cannot use an HTTP proxy)", apperr.ErrProxyConfig, v, httpsVar, allVar)
}

// Resolve turns trident's own proxy setting and the proxy environment into the
// one plan every consumer uses. It never returns nil and never fails: a problem
// is recorded and returned by Err, so that only commands which use the network
// abort. A nil getenv means an empty environment.
//
// A non-empty setting applies to https, http and DNS; the environment, NO_PROXY
// included, is then ignored and not validated. Otherwise https uses
// https_proxy/HTTPS_PROXY, then all_proxy/ALL_PROXY; http uses
// http_proxy/HTTP_PROXY, then the same all pair; DNS follows the https value.
// Only values a path actually uses are validated, and NO_PROXY (its pair and
// its entries) only when a proxy value is selected. NO_PROXY=* turns the proxy
// off. Two further rules abort: a lowercase/uppercase pair that disagrees, and
// an http proxy without an https one (https would go direct).
func Resolve(setting string, getenv func(string) string) *Settings {
	if setting != "" {
		return resolveSetting(setting)
	}
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	return resolveEnv(getenv)
}

func resolveSetting(setting string) *Settings {
	s := &Settings{viaSetting: true, httpsSelected: true, httpSelected: true}
	u, err := parseProxyURL(setting, settingSource)
	if err != nil {
		s.err = err
		return s
	}
	s.https = route{url: u, source: settingSource}
	s.http = s.https
	return s
}

func resolveEnv(getenv func(string) string) *Settings {
	s := &Settings{}

	no := pick(getenv, "no_proxy", "NO_PROXY", false)
	if no.err == nil && strings.TrimSpace(no.value) == "*" {
		// NO_PROXY=* says "connect directly on purpose": nothing else is
		// looked at, let alone validated.
		s.disabled = true
		s.noProxyValue = "*"
		return s
	}

	// Under CGI a client can inject HTTP_PROXY (the Proxy: header), so it
	// counts as unset before any rule or validation sees it. The lowercase
	// variable is the administrator's and stays.
	cgi := getenv("REQUEST_METHOD") != ""
	httpsPair := pick(getenv, "https_proxy", "HTTPS_PROXY", false)
	httpPair := pick(getenv, "http_proxy", "HTTP_PROXY", cgi)
	// The all pair is only consulted, and so only able to conflict, when it
	// supplies a value to at least one path.
	var allPair pair
	if !httpsPair.set || !httpPair.set {
		allPair = pick(getenv, "all_proxy", "ALL_PROXY", false)
	}

	useHTTPS, useHTTP := httpsPair, httpPair
	if !useHTTPS.set {
		useHTTPS = allPair
	}
	if !useHTTP.set {
		useHTTP = allPair
	}
	s.httpsSelected, s.httpSelected = useHTTPS.set, useHTTP.set

	// NO_PROXY only matters, and so is only checked (its pair for a conflict,
	// its entries for a parse error), when a proxy value is selected: with no
	// proxy there is nothing to exempt, and nothing a user could not honour.
	checkNoProxy := useHTTPS.set || useHTTP.set

	// One deferred error, the first of these classes in this order.
	var conflictErr, httpOnlyErr, valueErr, noProxyErr error
	pairs := []pair{httpsPair, httpPair, allPair}
	if checkNoProxy {
		pairs = append([]pair{no}, pairs...)
		s.noProxyErr = no.err
	}
	for _, p := range pairs {
		if p.err != nil {
			conflictErr = p.err
			break
		}
	}
	if !useHTTPS.set && httpPair.set && httpPair.err == nil {
		// The https chain is empty, so the http value is the http pair's.
		httpOnlyErr = httpOnlyError(httpPair.source)
	}

	// Only selected values are validated: a broken but shadowed variable is
	// ignored.
	if useHTTPS.err == nil && useHTTPS.value != "" {
		s.https.url, valueErr = parseProxyURL(useHTTPS.value, useHTTPS.source)
		s.https.source = useHTTPS.source
	}
	if useHTTP.err == nil && useHTTP.value != "" {
		if useHTTP.source == useHTTPS.source && useHTTP.value == useHTTPS.value {
			s.http = s.https
		} else {
			var err error
			s.http.url, err = parseProxyURL(useHTTP.value, useHTTP.source)
			s.http.source = useHTTP.source
			if valueErr == nil {
				valueErr = err
			}
		}
	}

	if no.err == nil {
		// The value in effect, shown as typed; parsed only when it matters.
		s.noProxyValue = no.value
		if checkNoProxy {
			var err error
			s.noProxy, err = parseNoProxy(no.value, no.source)
			if err != nil {
				noProxyErr, s.noProxyErr = err, err
			}
		}
	}

	for _, err := range []error{conflictErr, httpOnlyErr, valueErr, noProxyErr} {
		if err != nil {
			s.err = err
			break
		}
	}
	if s.err != nil {
		// Never keep a half-validated plan around.
		s.https, s.http, s.noProxy = route{}, route{}, noProxy{}
	}
	return s
}

// Err returns the deferred error of the run, or nil. It wraps
// apperr.ErrProxyConfig and never contains any part of a proxy value.
func (s *Settings) Err() error {
	return s.err
}

// ProxyFunc returns the proxy function for an HTTP transport: the validated
// proxy URL for a request, nil for a direct one. With a deferred error it
// returns that error for every request, so a broken setting can never turn
// into a direct connection. A request with scheme http uses the http proxy,
// every other scheme the https one (an unknown scheme errs toward proxying).
// NO_PROXY exempts a request only when the proxy comes from the environment;
// notify, if not nil, is called with the host of every request it exempts.
// The returned URLs are copies.
func (s *Settings) ProxyFunc(notify func(host string)) func(*http.Request) (*url.URL, error) {
	return func(r *http.Request) (*url.URL, error) {
		if s.err != nil {
			return nil, s.err
		}
		if s.disabled {
			return nil, nil
		}

		var scheme, host, port string
		if r != nil && r.URL != nil {
			scheme, host, port = r.URL.Scheme, r.URL.Hostname(), r.URL.Port()
		}
		rt, defaultPort := s.https, "443"
		if strings.EqualFold(scheme, "http") {
			rt, defaultPort = s.http, "80"
		}
		if rt.url == nil {
			return nil, nil
		}
		if port == "" {
			port = defaultPort
		}
		// A request without a host cannot be exempted: it is proxied.
		if !s.viaSetting && host != "" && s.noProxy.matches(host, port) {
			if notify != nil {
				notify(host)
			}
			return nil, nil
		}
		return cloneURL(rt.url), nil
	}
}

// Configured reports whether a proxy applies to https or http and NO_PROXY is
// not "*". With a deferred error it reports whether a proxy value was selected
// (valid or not), so that a caller that ignores Err still errs toward treating
// the run as proxied.
func (s *Settings) Configured() bool {
	return !s.disabled && (s.httpsSelected || s.httpSelected)
}

// HTTPSProxy returns a copy of the https proxy URL, or nil when https goes
// direct or the run has a deferred error.
func (s *Settings) HTTPSProxy() *url.URL {
	if s.err != nil || s.disabled {
		return nil
	}
	return cloneURL(s.https.url)
}

// DNSProxy returns a copy of the proxy DNS queries go through: the https proxy
// if it is a socks5 or socks5h one, else nil (nil also with a deferred error
// or NO_PROXY=*).
func (s *Settings) DNSProxy() *url.URL {
	if s.err != nil || s.disabled || s.https.url == nil {
		return nil
	}
	switch s.https.url.Scheme {
	case "socks5", "socks5h":
		return cloneURL(s.https.url)
	}
	return nil
}

// DNSUnproxied reports the case that is warned about: a proxy is configured
// for https but it is an http or https one, which DNS cannot use, so lookups
// use the system resolver. It is false with a deferred error or NO_PROXY=*.
func (s *Settings) DNSUnproxied() bool {
	if s.err != nil || s.disabled || s.https.url == nil {
		return false
	}
	switch s.https.url.Scheme {
	case "http", "https":
		return true
	}
	return false
}

// cloneURL returns a deep copy of u, or nil for nil.
func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	c := *u
	if u.User != nil {
		user := *u.User
		c.User = &user
	}
	return &c
}
