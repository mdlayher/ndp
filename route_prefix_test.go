package ndp

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"testing"
	"time"
)

func TestParseRouteInformationPrefixBits(t *testing.T) {
	address := netip.MustParseAddr("2001:db8:1234:56ff:ffff:ffff:ffff:ffff")
	for bits := 0; bits <= 128; bits++ {
		minimum := 1
		if bits > 0 {
			minimum = 2
		}
		if bits > 64 {
			minimum = 3
		}
		for length := minimum; length <= 3; length++ {
			t.Run(fmt.Sprintf("/%d/length%d", bits, length), func(t *testing.T) {
				want := netip.PrefixFrom(address, bits).Masked().Addr()
				wirePrefix := want.As16()
				// Receivers must ignore reserved bits beyond the prefix length.
				for bit := bits; bit < 128; bit++ {
					wirePrefix[bit/8] |= 1 << (7 - uint(bit%8))
				}
				// Complete ICMPv6 router advertisement with a route followed by an MTU.
				packet := []byte{134, 0, 0, 0, 64, 0, 0, 60, 0, 0, 0, 0, 0, 0, 0, 0}
				route := make([]byte, length*8)
				route[0], route[1], route[2], route[3] = 24, byte(length), byte(bits), byte(High)<<3
				binary.BigEndian.PutUint32(route[4:8], 60)
				copy(route[8:], wirePrefix[:])
				packet = append(packet, route...)
				packet = append(packet, 5, 1, 0, 0, 0, 0, 5, 220)
				parsed, err := ParseMessage(packet)
				if err != nil {
					t.Fatal(err)
				}
				ra, ok := parsed.(*RouterAdvertisement)
				if !ok || len(ra.Options) != 2 {
					t.Fatalf("unexpected advertisement: %+v", parsed)
				}
				got, ok := ra.Options[0].(*RouteInformation)
				if !ok {
					t.Fatalf("unexpected route option: %T", ra.Options[0])
				}
				if got.Prefix != want || got.PrefixLength != uint8(bits) || got.Preference != High || got.RouteLifetime != time.Minute {
					t.Errorf("route = %+v; want %s/%d, high preference and one minute", got, want, bits)
				}
				mtu, ok := ra.Options[1].(*MTU)
				if !ok || mtu.MTU != 1500 {
					t.Fatalf("following MTU = %+v", ra.Options[1])
				}
			})
		}
	}
}
