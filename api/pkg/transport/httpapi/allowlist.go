package httpapi

import (
	"context"
	"net/http"
	"net/netip"
	"strings"
)

// Allowlist rejects management requests that do not originate from an allowed
// network. It is the Go-side second layer described in plan section 5.5.
type Allowlist struct {
	networks []netip.Prefix
	trusted  []netip.Prefix
}

// NewAllowlist parses CIDR (or bare IP) entries. Loopback is always allowed so
// operators can probe the service locally; plan R6 keeps 127.0.0.1 reachable.
func NewAllowlist(networks, trusted []string) (*Allowlist, error) {
	allow := &Allowlist{}

	for _, entry := range []string{"127.0.0.0/8", "::1/128"} {
		prefix, err := parseNetwork(entry)
		if err != nil {
			return nil, err
		}

		allow.networks = append(allow.networks, prefix)
	}

	for _, entry := range networks {
		prefix, err := parseNetwork(entry)
		if err != nil {
			return nil, err
		}

		allow.networks = appendUniquePrefix(allow.networks, prefix)
	}

	for _, entry := range trusted {
		prefix, err := parseNetwork(entry)
		if err != nil {
			return nil, err
		}

		allow.trusted = appendUniquePrefix(allow.trusted, prefix)
	}

	return allow, nil
}

// Allows reports whether addr is inside one of the allowed networks.
func (a *Allowlist) Allows(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}

	addr = addr.Unmap()
	for _, prefix := range a.networks {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

// ResolveClient returns the effective client address.
//
// X-Forwarded-For is only honoured when the TCP peer is a configured trusted
// proxy. Otherwise a client-supplied header is ignored, which is what makes
// forged XFF headers harmless (plan section 5.5.1).
func (a *Allowlist) ResolveClient(remoteAddr string, header http.Header) (netip.Addr, bool) {
	peer, err := parseRemoteAddr(remoteAddr)
	if err != nil {
		return netip.Addr{}, false
	}

	if !a.isTrusted(peer) {
		return peer, true
	}

	chain := forwardedChain(header)
	client := peer

	for i := len(chain) - 1; i >= 0; i-- {
		client = chain[i]
		if !a.isTrusted(chain[i]) {
			break
		}
	}

	return client, true
}

func (a *Allowlist) isTrusted(addr netip.Addr) bool {
	addr = addr.Unmap()
	for _, prefix := range a.trusted {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

func forwardedChain(header http.Header) []netip.Addr {
	values := header.Values("X-Forwarded-For")
	if len(values) == 0 {
		return nil
	}

	chain := make([]netip.Addr, 0, len(values))

	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			addr, err := netip.ParseAddr(strings.TrimSpace(token))
			if err == nil {
				chain = append(chain, addr.Unmap())
			}
		}
	}

	return chain
}

func parseRemoteAddr(raw string) (netip.Addr, error) {
	if addrPort, err := netip.ParseAddrPort(raw); err == nil {
		return addrPort.Addr().Unmap(), nil
	}

	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Addr{}, err
	}

	return addr.Unmap(), nil
}

func parseNetwork(raw string) (netip.Prefix, error) {
	value := strings.TrimSpace(raw)

	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.Masked(), nil
	}

	addr, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Prefix{}, err
	}

	addr = addr.Unmap()

	return netip.PrefixFrom(addr, addr.BitLen()), nil
}

func appendUniquePrefix(prefixes []netip.Prefix, candidate netip.Prefix) []netip.Prefix {
	for _, existing := range prefixes {
		if existing == candidate {
			return prefixes
		}
	}

	return append(prefixes, candidate)
}

type clientIPKey struct{}

// ClientIPFromContext returns the address resolved by the allowlist
// middleware, or the zero value when it was never resolved.
func ClientIPFromContext(ctx context.Context) netip.Addr {
	addr, _ := ctx.Value(clientIPKey{}).(netip.Addr)
	return addr
}
