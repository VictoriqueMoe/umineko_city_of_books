package httpclient

import (
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPublicAddr(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want bool
	}{
		{name: "loopback v4", addr: "127.0.0.1", want: false},
		{name: "loopback v4 upper range", addr: "127.255.255.254", want: false},
		{name: "loopback v6", addr: "::1", want: false},
		{name: "rfc1918 ten dot", addr: "10.0.0.5", want: false},
		{name: "rfc1918 172.16 lower bound", addr: "172.16.0.1", want: false},
		{name: "rfc1918 172.31 upper bound", addr: "172.31.255.254", want: false},
		{name: "rfc1918 192.168", addr: "192.168.1.1", want: false},
		{name: "link local v4 metadata endpoint", addr: "169.254.169.254", want: false},
		{name: "tencent metadata endpoint", addr: "169.254.0.23", want: false},
		{name: "link local v6", addr: "fe80::1", want: false},
		{name: "unique local fc00", addr: "fc00::1", want: false},
		{name: "unique local fd00", addr: "fd12:3456:789a::1", want: false},
		{name: "aws metadata endpoint over v6", addr: "fd00:ec2::254", want: false},
		{name: "ipv4 mapped private", addr: "::ffff:10.0.0.1", want: false},
		{name: "ipv4 mapped loopback", addr: "::ffff:127.0.0.1", want: false},
		{name: "ipv4 mapped link local", addr: "::ffff:169.254.169.254", want: false},
		{name: "ipv4 mapped link local in hex", addr: "::ffff:a9fe:a9fe", want: false},
		{name: "ipv4 mapped shared address space", addr: "::ffff:100.100.100.200", want: false},
		{name: "unspecified v4", addr: "0.0.0.0", want: false},
		{name: "unspecified v6", addr: "::", want: false},
		{name: "this network", addr: "0.1.2.3", want: false},
		{name: "multicast v4", addr: "224.0.0.1", want: false},
		{name: "multicast v6", addr: "ff02::1", want: false},
		{name: "alibaba metadata endpoint", addr: "100.100.100.200", want: false},
		{name: "shared address space lower bound", addr: "100.64.0.1", want: false},
		{name: "shared address space upper bound", addr: "100.127.255.254", want: false},
		{name: "ietf protocol assignments", addr: "192.0.0.8", want: false},
		{name: "documentation test net 1", addr: "192.0.2.1", want: false},
		{name: "benchmarking lower bound", addr: "198.18.0.1", want: false},
		{name: "benchmarking upper bound", addr: "198.19.255.254", want: false},
		{name: "documentation test net 2", addr: "198.51.100.1", want: false},
		{name: "documentation test net 3", addr: "203.0.113.1", want: false},
		{name: "reserved class e", addr: "240.0.0.1", want: false},
		{name: "limited broadcast", addr: "255.255.255.255", want: false},
		{name: "ipv4 compatible link local", addr: "::a9fe:a9fe", want: false},
		{name: "nat64 well known wrapping link local", addr: "64:ff9b::a9fe:a9fe", want: false},
		{name: "nat64 local use", addr: "64:ff9b:1::1", want: false},
		{name: "discard only", addr: "100::1", want: false},
		{name: "teredo", addr: "2001:0:4136:e378::1", want: false},
		{name: "documentation v6", addr: "2001:db8::1", want: false},
		{name: "6to4 wrapping link local", addr: "2002:a9fe:a9fe::1", want: false},
		{name: "documentation v6 3fff", addr: "3fff::1", want: false},
		{name: "srv6 sids", addr: "5f00::1", want: false},
		{name: "deprecated site local", addr: "fec0::1", want: false},
		{name: "deprecated site local with a zone", addr: "fec0::1%eth0", want: false},
		{name: "public v4 cloudflare", addr: "1.1.1.1", want: true},
		{name: "public v4 google", addr: "8.8.8.8", want: true},
		{name: "public v4 just outside rfc1918", addr: "172.32.0.1", want: true},
		{name: "public v4 just below shared address space", addr: "100.63.255.255", want: true},
		{name: "public v4 just above shared address space", addr: "100.128.0.1", want: true},
		{name: "public v6", addr: "2606:4700:4700::1111", want: true},
		{name: "public v6 under 2001 outside protocol assignments", addr: "2001:4860:4860::8888", want: true},
		{name: "public ipv4 mapped", addr: "::ffff:8.8.8.8", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			addr, err := netip.ParseAddr(tc.addr)
			require.NoError(t, err)

			// when
			got := isPublicAddr(addr)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsPublicAddr_InvalidAddrIsBlocked(t *testing.T) {
	// given
	var addr netip.Addr

	// when
	got := isPublicAddr(addr)

	// then
	assert.False(t, got)
}

func TestBlockNonPublicAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		blocked bool
	}{
		{name: "loopback v4 with port", address: "127.0.0.1:8080", blocked: true},
		{name: "loopback v6 with port", address: "[::1]:8080", blocked: true},
		{name: "docker bridge host", address: "172.17.0.1:5432", blocked: true},
		{name: "metadata endpoint", address: "169.254.169.254:80", blocked: true},
		{name: "alibaba metadata endpoint", address: "100.100.100.200:80", blocked: true},
		{name: "ipv4 mapped private with port", address: "[::ffff:192.168.0.10]:443", blocked: true},
		{name: "zoned site local with port", address: "[fec0::1%eth0]:80", blocked: true},
		{name: "unparsable address", address: "not-an-address", blocked: true},
		{name: "unresolved hostname", address: "internal-db:5432", blocked: true},
		{name: "public v4 with port", address: "1.1.1.1:443", blocked: false},
		{name: "public v6 with port", address: "[2606:4700:4700::1111]:443", blocked: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			address := tc.address

			// when
			err := blockNonPublicAddress("tcp", address, nil)

			// then
			if tc.blocked {
				assert.ErrorIs(t, err, ErrBlockedAddress)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestWithPublicAddressesOnly_RefusesANonPublicTarget(t *testing.T) {
	// given
	client := New(time.Second, WithPublicAddressesOnly())

	// when
	_, err := client.Do(t.Context(), Request{Method: http.MethodGet, URL: "http://127.0.0.1:1/"})

	// then
	assert.ErrorIs(t, err, ErrBlockedAddress)
}

func TestWithPublicAddressesOnly_BoundsTheConnectionAndIgnoresProxies(t *testing.T) {
	// given
	timeout := 5 * time.Second

	// when
	client := New(timeout, WithPublicAddressesOnly())

	// then
	transport, ok := client.HTTP().Transport.(*http.Transport)
	require.True(t, ok)
	assert.Equal(t, timeout, transport.TLSHandshakeTimeout)
	assert.Equal(t, int64(http.DefaultMaxHeaderBytes), transport.MaxResponseHeaderBytes)
	assert.Nil(t, transport.Proxy)
}
