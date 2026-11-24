package apperr

import "errors"

// ErrInvalidInput is returned by any service when the provided input fails validation.
// Use errors.Is(err, apperr.ErrInvalidInput) to detect validation failures uniformly
// across all services.
var ErrInvalidInput = errors.New("invalid input")

// ErrRequestFailed is returned by any HTTP-based service when the request fails at the
// transport level or the server responds with a non-2xx status code.
// Use errors.Is(err, apperr.ErrRequestFailed) to detect request failures uniformly
// across all services.
var ErrRequestFailed = errors.New("request failed")

// ErrPAPBlocked is returned when a service's PAP level exceeds the user-defined limit.
var ErrPAPBlocked = errors.New("PAP limit exceeded")

// ErrProxyBypass is returned when a connection would leave the configured proxy:
// the HTTP client refuses to dial a host directly that the proxy is meant to
// reach. Retrying cannot help, so it is never retried.
var ErrProxyBypass = errors.New("refusing direct connection while a proxy is configured")

// ErrUnsupportedProxy is returned when a client cannot speak the configured
// proxy's scheme, such as socks4:// for DNS-over-HTTPS. Retrying cannot help,
// so it is never retried.
var ErrUnsupportedProxy = errors.New("proxy scheme not supported")

// ErrProxyConfig is returned when a proxy setting (--proxy, the config file,
// TRIDENT_PROXY or a proxy environment variable) is not a usable proxy URL.
// The message names the setting and the reason, never the value. Retrying
// cannot help, so it is never retried.
var ErrProxyConfig = errors.New("invalid proxy configuration")

// RequestFailure returns the sentinel a failed request's error is wrapped
// with: the proxy sentinel err carries (ErrProxyBypass, ErrUnsupportedProxy or
// ErrProxyConfig), so that callers can still tell that the proxy could not be
// honoured (a bulk run stops at it), and ErrRequestFailed otherwise. Wrap
// with it as the single %w and keep err as %v:
//
//	fmt.Errorf("%w: crt.sh request for %q: %v", apperr.RequestFailure(err), domain, err)
func RequestFailure(err error) error {
	for _, sentinel := range []error{ErrProxyBypass, ErrUnsupportedProxy, ErrProxyConfig} {
		if errors.Is(err, sentinel) {
			return sentinel
		}
	}
	return ErrRequestFailed
}
