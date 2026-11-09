package httpclient

import (
	"net"
	"net/http"
	"net/netip"
	"slices"
	"syscall"
	"time"
)

var (
	nonPublicPrefixes = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("::/96"),
		netip.MustParsePrefix("64:ff9b::/96"),
		netip.MustParsePrefix("64:ff9b:1::/48"),
		netip.MustParsePrefix("100::/64"),
		netip.MustParsePrefix("2001::/23"),
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("2002::/16"),
		netip.MustParsePrefix("3fff::/20"),
		netip.MustParsePrefix("5f00::/16"),
		netip.MustParsePrefix("fec0::/10"),
	}
)

func WithPublicAddressesOnly() Option {
	return func(client *http.Client) {
		client.Transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: client.Timeout,
				Control: blockNonPublicAddress,
			}).DialContext,
			TLSHandshakeTimeout:    client.Timeout,
			MaxResponseHeaderBytes: http.DefaultMaxHeaderBytes,
			ForceAttemptHTTP2:      true,
			MaxIdleConns:           100,
			IdleConnTimeout:        90 * time.Second,
		}
	}
}

func blockNonPublicAddress(_, address string, _ syscall.RawConn) error {
	addrPort, err := netip.ParseAddrPort(address)
	if err != nil {
		return ErrBlockedAddress
	}

	if !isPublicAddr(addrPort.Addr()) {
		return ErrBlockedAddress
	}

	return nil
}

func isPublicAddr(addr netip.Addr) bool {
	addr = addr.Unmap().WithZone("")

	if !addr.IsValid() {
		return false
	}

	switch {
	case addr.IsLoopback(),
		addr.IsPrivate(),
		addr.IsUnspecified(),
		addr.IsLinkLocalUnicast(),
		addr.IsLinkLocalMulticast(),
		addr.IsMulticast():
		return false
	}

	return !slices.ContainsFunc(nonPublicPrefixes, func(prefix netip.Prefix) bool {
		return prefix.Contains(addr)
	})
}
