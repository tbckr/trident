package resolver

import (
	"context"
	"fmt"
	"net"
	"strings"

	"golang.org/x/net/proxy"
)

// NewResolver returns a *net.Resolver appropriate for the given proxy URL.
//
// When proxyURL is empty or its scheme is not "socks5", the standard system
// resolver is returned (nil Dial field — Go uses the platform resolver).
//
// When proxyURL is a socks5:// URL, DNS queries are tunnelled through the
// SOCKS5 proxy using DNS-over-TCP, preventing DNS leaks to the local ISP.
func NewResolver(proxyURL string) (*net.Resolver, error) {
	if proxyURL == "" || !strings.HasPrefix(proxyURL, "socks5://") {
		return &net.Resolver{}, nil
	}

	// Strip the scheme to get host:port.
	host := strings.TrimPrefix(proxyURL, "socks5://")

	dialer, err := proxy.SOCKS5("tcp", host, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("creating SOCKS5 dialer for DNS: %w", err)
	}

	// proxy.SOCKS5 returns a ContextDialer — type-assert to get DialContext.
	ctxDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("SOCKS5 dialer does not implement ContextDialer")
	}

	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return ctxDialer.DialContext(ctx, "tcp", address)
		},
	}, nil
}
