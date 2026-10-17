package dronebl

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engine"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type deps struct {
	settings *settings.MockService
	resolver *MockResolver
}

func newChecker(t *testing.T) (*Checker, *deps) {
	t.Helper()

	d := &deps{
		settings: settings.NewMockService(t),
		resolver: NewMockResolver(t),
	}
	d.settings.EXPECT().Get(mock.Anything, config.SettingDroneBLAllowlist).Return("").Maybe()
	d.settings.EXPECT().Get(mock.Anything, config.SettingDroneBLIgnoredClasses).Return("").Maybe()

	return New(d.settings, cache.NewManager(engines.NewInMemory(0)), d.resolver, nil), d
}

func listed(class byte) []netip.Addr {
	return []netip.Addr{netip.AddrFrom4([4]byte{127, 0, 0, class})}
}

func notFound() error {
	return &net.DNSError{Err: "no such host", IsNotFound: true}
}

func TestCheck_ListedAddressCarriesItsClass(t *testing.T) {
	// given an address DroneBL answers for
	checker, d := newChecker(t)
	d.resolver.EXPECT().
		LookupNetIP(mock.Anything, "ip4", "165.41.114.190.dnsbl.dronebl.org").
		Return(listed(13), nil).
		Once()

	// when
	verdict := checker.Check(context.Background(), "190.114.41.165")

	// then
	assert.True(t, verdict.Listed)
	assert.Equal(t, []int{13}, verdict.Classes)
}

func TestCheck_CleanAddressIsNotListed(t *testing.T) {
	// given DroneBL answers NXDOMAIN, which is how it reports a clean address
	checker, d := newChecker(t)
	d.resolver.EXPECT().LookupNetIP(mock.Anything, "ip4", mock.Anything).Return(nil, notFound()).Once()

	// when
	verdict := checker.Check(context.Background(), "8.8.8.8")

	// then
	assert.False(t, verdict.Listed)
}

func TestCheck_IPv6IsQueriedInNibbleFormat(t *testing.T) {
	// given a v6 member address, since prod hands out v6
	checker, d := newChecker(t)
	d.resolver.EXPECT().
		LookupNetIP(mock.Anything, "ip4", "4.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.5.1.0.4.5.1.0.0.7.8.3.0.0.0.6.2.dnsbl.dronebl.org").
		Return(listed(17), nil).
		Once()

	// when
	verdict := checker.Check(context.Background(), "2600:387:15:4015::4")

	// then
	assert.True(t, verdict.Listed)
	assert.Equal(t, []int{17}, verdict.Classes)
}

func TestCheck_DNSFailureFailsOpen(t *testing.T) {
	// given the resolver itself is broken, not the address
	checker, d := newChecker(t)
	d.resolver.EXPECT().LookupNetIP(mock.Anything, "ip4", mock.Anything).Return(nil, errors.New("server misbehaving")).Once()

	// when
	verdict := checker.Check(context.Background(), "203.0.113.9")

	// then an unreachable blocklist must never lock the site down
	assert.False(t, verdict.Listed)
}

func TestCheck_VerdictLifetimeDependsOnWhetherTheLookupWorked(t *testing.T) {
	tests := []struct {
		name      string
		lookupErr error
		wantTTL   time.Duration
	}{
		{name: "an answered lookup holds for the verdict ttl", lookupErr: notFound(), wantTTL: cache.DroneBL.TTL},
		{name: "a failed lookup holds only briefly so the address is checked again soon", lookupErr: errors.New("server misbehaving"), wantTTL: failureTTL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given an empty store that expects the verdict in the slot Check reads
			key := cache.DroneBL.Key("203.0.113.9").String()

			store := engine.NewMockEngine(t)
			store.EXPECT().Enabled().Return(true).Maybe()
			store.EXPECT().Get(mock.Anything, key).Return(nil, cache.ErrMiss).Once()
			store.EXPECT().Set(mock.Anything, key, []byte(`{"listed":false}`), tt.wantTTL).Return(nil).Once()

			resolver := NewMockResolver(t)
			resolver.EXPECT().LookupNetIP(mock.Anything, "ip4", mock.Anything).Return(nil, tt.lookupErr).Once()

			checker := New(settings.NewMockService(t), cache.NewManager(store), resolver, nil)

			// when
			verdict := checker.Check(context.Background(), "203.0.113.9")

			// then the store saw the expected lifetime, which the mock enforces
			assert.False(t, verdict.Listed)
		})
	}
}

func TestCheck_VerdictIsCachedSoEveryRequestIsNotALookup(t *testing.T) {
	// given one address asked for twice
	checker, d := newChecker(t)
	d.resolver.EXPECT().LookupNetIP(mock.Anything, "ip4", mock.Anything).Return(listed(9), nil).Once()

	// when
	first := checker.Check(context.Background(), "89.187.168.238")
	second := checker.Check(context.Background(), "89.187.168.238")

	// then the resolver is consulted once, which Once() enforces
	assert.True(t, first.Listed)
	assert.True(t, second.Listed)
}

func TestCheck_UnroutableAddressIsNeverSentToDroneBL(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{name: "private", ip: "10.0.0.4"},
		{name: "loopback", ip: "127.0.0.1"},
		{name: "junk", ip: "not-an-ip"},
		{name: "empty", ip: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a resolver that must never be called
			checker, _ := newChecker(t)

			// when
			verdict := checker.Check(context.Background(), tc.ip)

			// then no expectation was registered, so any lookup would fail the test
			assert.False(t, verdict.Listed)
		})
	}
}

func TestBlocked_IgnoredClassesLetAMemberThrough(t *testing.T) {
	tests := []struct {
		name    string
		class   byte
		ignored string
		want    bool
	}{
		{name: "no ignore list blocks any listing", class: 13, ignored: "", want: true},
		{name: "the ignored class does not block", class: 13, ignored: "13", want: false},
		{name: "another class still blocks", class: 9, ignored: "13", want: true},
		{name: "several ignored classes", class: 18, ignored: "13, 18", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			d := &deps{settings: settings.NewMockService(t), resolver: NewMockResolver(t)}
			d.settings.EXPECT().Get(mock.Anything, config.SettingDroneBLIgnoredClasses).Return(tc.ignored).Maybe()
			checker := New(d.settings, cache.NewManager(engines.NewInMemory(0)), d.resolver, nil)
			d.resolver.EXPECT().LookupNetIP(mock.Anything, "ip4", mock.Anything).Return(listed(tc.class), nil).Once()

			// when
			verdict, blocked := checker.Blocked(context.Background(), "203.0.113.44")

			// then
			require.True(t, verdict.Listed)
			assert.Equal(t, tc.want, blocked)
		})
	}
}

func TestAllowlisted(t *testing.T) {
	// given an operator who has rescued their own address
	d := &deps{settings: settings.NewMockService(t), resolver: NewMockResolver(t)}
	d.settings.EXPECT().Get(mock.Anything, config.SettingDroneBLAllowlist).Return("203.0.113.0/24").Maybe()
	checker := New(d.settings, cache.NewManager(engines.NewInMemory(0)), d.resolver, nil)

	// then the resolver is never consulted for an allowlisted address
	assert.True(t, checker.Allowlisted(context.Background(), "203.0.113.7"))
	assert.False(t, checker.Allowlisted(context.Background(), "198.51.100.7"))
}

func TestEnabled(t *testing.T) {
	// given
	d := &deps{settings: settings.NewMockService(t), resolver: NewMockResolver(t)}
	d.settings.EXPECT().GetBool(mock.Anything, config.SettingDroneBLEnabled).Return(false).Once()
	checker := New(d.settings, cache.NewManager(engines.NewInMemory(0)), d.resolver, nil)

	// then
	assert.False(t, checker.Enabled(context.Background()))
}
