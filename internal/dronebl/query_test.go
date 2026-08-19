package dronebl

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryName(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want string
		ok   bool
	}{
		{
			name: "ipv4 reverses the octets",
			ip:   "127.0.0.2",
			want: "2.0.0.127.dnsbl.dronebl.org",
			ok:   false,
		},
		{
			name: "a routable ipv4 address",
			ip:   "1.1.1.1",
			want: "1.1.1.1.dnsbl.dronebl.org",
			ok:   true,
		},
		{
			name: "an asymmetric ipv4 address proves the order",
			ip:   "190.114.41.165",
			want: "165.41.114.190.dnsbl.dronebl.org",
			ok:   true,
		},
		{
			name: "ipv6 expands to reversed nibbles",
			ip:   "2001:4860:4860::8888",
			want: "8.8.8.8.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.6.8.4.0.6.8.4.1.0.0.2.dnsbl.dronebl.org",
			ok:   true,
		},
		{
			name: "a real listed ipv6 member address",
			ip:   "2600:387:15:4015::4",
			want: "4.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.5.1.0.4.5.1.0.0.7.8.3.0.0.0.6.2.dnsbl.dronebl.org",
			ok:   true,
		},
		{
			name: "an ipv4 mapped ipv6 address is treated as ipv4",
			ip:   "::ffff:8.8.4.4",
			want: "4.4.8.8.dnsbl.dronebl.org",
			ok:   true,
		},
		{
			name: "a private address is never sent to a third party",
			ip:   "192.168.1.5",
			ok:   false,
		},
		{
			name: "loopback is skipped",
			ip:   "127.0.0.1",
			ok:   false,
		},
		{
			name: "an ipv6 link local address is skipped",
			ip:   "fe80::1",
			ok:   false,
		},
		{
			name: "junk is skipped rather than queried",
			ip:   "not-an-ip",
			ok:   false,
		},
		{
			name: "an empty address is skipped",
			ip:   "",
			ok:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a client address

			// when
			got, ok := queryName(tc.ip)

			// then
			if !tc.ok {
				assert.False(t, ok)
				return
			}

			require.True(t, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestQueryName_IPv6HasOneLabelPerNibble(t *testing.T) {
	// given any ipv6 address
	name, ok := queryName("2402:800:6378:8672:785a:5340:9063:fa81")
	require.True(t, ok)

	// then RFC 5782 requires one label per nibble, which is what DroneBL answered when probed
	assert.Equal(t, 34, strings.Count(name, "."))
}

func TestClassesFrom(t *testing.T) {
	tests := []struct {
		name    string
		answers []string
		want    []int
	}{
		{
			name:    "a real listing yields its class",
			answers: []string{"127.0.0.13"},
			want:    []int{13},
		},
		{
			name:    "the rfc test entry is not a real listing",
			answers: []string{"127.0.0.1"},
			want:    []int{},
		},
		{
			name:    "several classes are all reported",
			answers: []string{"127.0.0.3", "127.0.0.9"},
			want:    []int{3, 9},
		},
		{
			name:    "an answer outside 127/8 is ignored",
			answers: []string{"10.0.0.5"},
			want:    []int{},
		},
		{
			name:    "no answers means clean",
			answers: nil,
			want:    []int{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			addrs := make([]netip.Addr, 0, len(tc.answers))
			for _, raw := range tc.answers {
				addrs = append(addrs, netip.MustParseAddr(raw))
			}

			// when
			got := classesFrom(addrs)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseClassFilter(t *testing.T) {
	// given
	filter := parseClassFilter(" 13 , 18 ,nonsense, 0 , 999 ")

	// then only sane classes survive, and junk never widens the block
	require.NotNil(t, filter)
	assert.True(t, filter[13])
	assert.True(t, filter[18])
	assert.Len(t, filter, 2)

	// and an empty setting ignores nothing, so every listing still blocks
	assert.Empty(t, parseClassFilter("   "))
	assert.Empty(t, parseClassFilter("nonsense"))
}

func TestAllowlist(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		ip   string
		want bool
	}{
		{name: "a bare ipv4 address matches itself", raw: "203.0.113.4", ip: "203.0.113.4", want: true},
		{name: "a bare address does not match a neighbour", raw: "203.0.113.4", ip: "203.0.113.5", want: false},
		{name: "a cidr covers its range", raw: "203.0.113.0/24", ip: "203.0.113.99", want: true},
		{name: "a cidr excludes outside its range", raw: "203.0.113.0/24", ip: "203.0.114.1", want: false},
		{name: "an ipv6 prefix covers the whole subnet", raw: "2600:387:15:4015::/64", ip: "2600:387:15:4015::4", want: true},
		{name: "several entries are all honoured", raw: "10.0.0.1, 203.0.113.0/24", ip: "203.0.113.7", want: true},
		{name: "an empty allowlist matches nothing", raw: "", ip: "203.0.113.4", want: false},
		{name: "junk entries are skipped without matching", raw: "not-a-cidr", ip: "203.0.113.4", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			prefixes := parseAllowlist(tc.raw)

			// when
			got := allowlisted(prefixes, tc.ip)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}
