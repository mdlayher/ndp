package ndp

import (
	"net"
	"net/netip"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_chooseAddr(t *testing.T) {
	// Assumed zone for all tests.
	const zone = "eth0"

	var (
		ip4 = net.IPv4(192, 168, 1, 1).To4()
		ip6 = mustIPv6("2001:db8::1000")

		gua = mustIPv6("2001:db8::1")
		ula = mustIPv6("fc00::1")
		lla = mustIPv6("fe80::1")
	)

	addrs := []net.Addr{
		// Ignore non-IP addresses.
		&net.TCPAddr{IP: gua},

		&net.IPNet{IP: ip4},
		&net.IPNet{IP: ula},
		&net.IPNet{IP: lla},

		// The second GUA IPv6 address should only be found when
		// Addr specifies it explicitly.
		&net.IPNet{IP: gua},
		&net.IPNet{IP: ip6},
	}

	tests := []struct {
		name  string
		addrs []net.Addr
		addr  Addr
		ip    netip.Addr
		ok    bool
	}{
		{
			name: "empty",
		},
		{
			name: "IPv4 Addr",
			addr: Addr(ip4.String()),
		},
		{
			name: "no IPv6 addresses",
			addrs: []net.Addr{
				&net.IPNet{
					IP: ip4,
				},
			},
			addr: LinkLocal,
		},
		{
			name: "ok, unspecified",
			ip:   netip.IPv6Unspecified(),
			addr: Unspecified,
			ok:   true,
		},
		{
			name:  "ok, GUA",
			addrs: addrs,
			ip:    netip.MustParseAddr("2001:db8::1"),
			addr:  Global,
			ok:    true,
		},
		{
			name:  "ok, ULA",
			addrs: addrs,
			ip:    netip.MustParseAddr("fc00::1"),
			addr:  UniqueLocal,
			ok:    true,
		},
		{
			name:  "ok, LLA",
			addrs: addrs,
			ip:    netip.MustParseAddr("fe80::1"),
			addr:  LinkLocal,
			ok:    true,
		},
		{
			name:  "ok, arbitrary",
			addrs: addrs,
			ip:    netip.MustParseAddr("2001:db8::1000"),
			addr:  Addr(ip6.String()),
			ok:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipa, err := chooseAddr(tt.addrs, zone, tt.addr)

			if err != nil && tt.ok {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && !tt.ok {
				t.Fatal("expected an error, but none occurred")
			}
			if err != nil {
				t.Logf("OK error: %v", err)
				return
			}

			ttipa := tt.ip.WithZone(zone)
			if diff := cmp.Diff(ttipa, ipa, cmp.Comparer(addrEqual)); diff != "" {
				t.Fatalf("unexpected IPv6 address (-want +got):\n%s", diff)
			}
		})
	}
}

// MustIPv6 parses s as a valid IPv6 address, or it panics.
func mustIPv6(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil || ip.To4() != nil {
		panicf("invalid IPv6 address: %q", s)
	}

	return ip
}
