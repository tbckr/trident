package resolver

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/net/proxy"
)

// NewResolver returns a *net.Resolver appropriate for the given proxy URL.
//
// When proxyURL is empty, the ALL_PROXY environment variable is consulted via
// proxy.FromEnvironment(). If it resolves to a SOCKS5 dialer, DNS queries are
// tunnelled through it. HTTP/HTTPS proxies are intentionally ignored for DNS
// (only SOCKS5 can proxy raw TCP DNS traffic). If no usable proxy is found,
// the standard system resolver is returned (nil Dial field).
//
// When proxyURL is a non-empty non-socks5 URL, the standard system resolver
// is returned.
//
// When proxyURL is a socks5:// URL, DNS queries are tunnelled through the
// SOCKS5 proxy using DNS-over-TCP, preventing DNS leaks to the local ISP.
func NewResolver(proxyURL string) (*net.Resolver, error) {
	if proxyURL == "" {
		// Honour ALL_PROXY / all_proxy env var. Only SOCKS5 can proxy raw TCP
		// DNS traffic; HTTP/HTTPS proxies are intentionally ignored.
		allProxy := os.Getenv("ALL_PROXY")
		if allProxy == "" {
			allProxy = os.Getenv("all_proxy")
		}
		if host, ok := strings.CutPrefix(allProxy, "socks5://"); ok {
			dialer, err := proxy.SOCKS5("tcp", host, nil, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("creating SOCKS5 dialer for DNS from ALL_PROXY: %w", err)
			}
			if cd, ok := dialer.(proxy.ContextDialer); ok {
				return newSocks5Resolver(cd), nil
			}
		}
		return &net.Resolver{}, nil
	}

	host, ok := strings.CutPrefix(proxyURL, "socks5://")
	if !ok {
		return &net.Resolver{}, nil
	}

	dialer, err := proxy.SOCKS5("tcp", host, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("creating SOCKS5 dialer for DNS: %w", err)
	}

	// proxy.SOCKS5 returns a ContextDialer — type-assert to get DialContext.
	ctxDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("SOCKS5 dialer does not implement ContextDialer")
	}

	return newSocks5Resolver(ctxDialer), nil
}

func newSocks5Resolver(cd proxy.ContextDialer) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return cd.DialContext(ctx, "tcp", address)
		},
	}
}
