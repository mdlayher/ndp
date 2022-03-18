package ndp

// Package ndp_test not used because we need access to direct option marshaling
// and unmarshaling functions.

import (
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/ndp/internal/ndptest"
)

// An optionSub is a sub-test structure for Option marshal/unmarshal tests.
type optionSub struct {
	name string
	os   []Option
	bs   [][]byte
	ok   bool
}

func TestOptionMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		subs []optionSub
	}{
		{
			name: "raw option",
			subs: roTests(),
		},
		{
			name: "link layer address",
			subs: llaTests(),
		},
		{
			name: "MTU",
			subs: []optionSub{{
				name: "ok",
				os: []Option{
					NewMTU(1500),
				},
				bs: [][]byte{
					{0x05, 0x01, 0x00, 0x00},
					{0x00, 0x00, 0x05, 0xdc},
				},
				ok: true,
			}},
		},
		{
			name: "prefix information",
			subs: piTests(),
		},
		{
			name: "route information",
			subs: riTests(),
		},
		{
			name: "recursive DNS servers",
			subs: rdnssTests(),
		},
		{
			name: "DNS search list",
			subs: dnsslTests(),
		},
		{
			name: "captive portal",
			subs: cpTests(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, st := range tt.subs {
				t.Run(st.name, func(t *testing.T) {
					b, err := marshalOptions(st.os)

					if err != nil && st.ok {
						t.Fatalf("unexpected error: %v", err)
					}
					if err == nil && !st.ok {
						t.Fatal("expected an error, but none occurred")
					}
					if err != nil {
						t.Logf("OK error: %v", err)
						return
					}

					ttb := ndptest.Merge(st.bs)
					if diff := cmp.Diff(ttb, b); diff != "" {
						t.Fatalf("unexpected options bytes (-want +got):\n%s", diff)
					}

					got, err := parseOptions(b)
					if err != nil {
						t.Fatalf("failed to unmarshal options: %v", err)
					}

					if diff := cmp.Diff(st.os, got, cmp.Comparer(addrEqual)); diff != "" {
						t.Fatalf("unexpected options (-want +got):\n%s", diff)
					}
				})
			}
		})
	}
}

func TestOptionUnmarshalError(t *testing.T) {
	type sub struct {
		name string
		bs   [][]byte
	}

	tests := []struct {
		name string
		o    Option
		subs []sub
	}{
		{
			name: "raw option",
			o:    &RawOption{},
			subs: []sub{
				{
					name: "short",
					bs:   [][]byte{{0x01}},
				},
				{
					name: "misleading length",
					bs:   [][]byte{{0x10, 0x10}},
				},
			},
		},
		{
			name: "link layer address",
			o:    &LinkLayerAddress{},
			subs: []sub{
				{
					name: "short",
					bs:   [][]byte{{0x01, 0x01, 0xff}},
				},
				{
					name: "invalid direction",
					bs: [][]byte{
						{0x10, 0x01},
						ndptest.MAC,
					},
				},
				{
					name: "long",
					bs: [][]byte{
						{0x01, 0x02},
						ndptest.Zero(16),
					},
				},
			},
		},
		{
			name: "mtu",
			o:    new(MTU),
			subs: []sub{
				{
					name: "short",
					bs:   [][]byte{{0x01}},
				},
			},
		},
		{
			name: "prefix information",
			o:    &PrefixInformation{},
			subs: []sub{
				{
					name: "short",
					bs:   [][]byte{{0x01}},
				},
			},
		},
		{
			name: "route information",
			o:    &RouteInformation{},
			subs: []sub{
				{
					name: "short",
					bs:   [][]byte{{0x01}},
				},

				{
					name: "bad /0",
					bs: [][]byte{
						// Length must be 1-3.
						{24, 0x04},
						ndptest.Zero(30),
					},
				},
				{
					name: "bad /64",
					bs: [][]byte{
						// Length must be 2-3.
						{24, 0x01},
						{64, 0x04},
						{0x00, 0x00, 0x00, 0xff},
					},
				},
				{
					name: "bad /96",
					bs: [][]byte{
						// Length must be 3.
						{24, 0x04},
						{96, 0x04},
						ndptest.Zero(28),
					},
				},
				{
					name: "bad /255",
					bs: [][]byte{
						{24, 0x01},
						// Invalid IPv6 prefix.
						{0xff, 0x00},
						ndptest.Zero(4),
					},
				},
				{
					name: "bad preference",
					bs: [][]byte{
						{24, 0x01},
						// Reserved preference.
						{0, 0x10},
						ndptest.Zero(4),
					},
				},
			},
		},
		{
			name: "rdnss",
			o:    &RecursiveDNSServer{},
			subs: []sub{
				{
					name: "no servers",
					bs: [][]byte{
						{25, 1},
						// Reserved.
						{0x00, 0x00},
						// Lifetime.
						ndptest.Zero(4),
						// No servers.
					},
				},
				{
					name: "bad first server",
					bs: [][]byte{
						{25, 2},
						// Reserved.
						{0x00, 0x00},
						// Lifetime.
						ndptest.Zero(4),
						// First server, half an IPv6 address.
						ndptest.Zero(8),
					},
				},
				{
					name: "bad second server",
					bs: [][]byte{
						{25, 4},
						// Reserved.
						{0x00, 0x00},
						// Lifetime.
						ndptest.Zero(4),
						// First server.
						ndptest.Zero(16),
						// Second server, half an IPv6 address.
						ndptest.Zero(8),
					},
				},
			},
		},
		{
			name: "dnssl",
			o:    &DNSSearchList{},
			subs: []sub{
				{
					name: "no domains",
					bs: [][]byte{
						{31, 1},
						// Reserved.
						{0x00, 0x00},
						// Lifetime.
						ndptest.Zero(4),
						// No domains.
					},
				},
				{
					name: "misleading length",
					bs: [][]byte{
						{31, 2},
						// Reserved.
						{0x00, 0x00},
						// Lifetime.
						ndptest.Zero(4),
						// Length misleading.
						{0xff},
						ndptest.Zero(7),
					},
				},
				{
					name: "no room for null terminator",
					bs: [][]byte{
						{31, 2},
						// Reserved.
						{0x00, 0x00},
						// Lifetime.
						ndptest.Zero(4),
						// Length leaves no room for null terminator.
						{7},
						ndptest.Zero(7),
					},
				},
				{
					name: "no domains, padded",
					bs: [][]byte{
						{31, 2},
						// Reserved.
						{0x00, 0x00},
						// Lifetime.
						ndptest.Zero(4),
						// No domains.
						ndptest.Zero(8),
					},
				},
			},
		},
		{
			name: "captive portal",
			o:    new(CaptivePortal),
			subs: []sub{
				{
					name: "null URI",
					bs: [][]byte{
						{37, 1},
						// URI.
						ndptest.Zero(6),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, st := range tt.subs {
				t.Run(st.name, func(t *testing.T) {
					err := tt.o.unmarshal(ndptest.Merge(st.bs))

					if err == nil {
						t.Fatal("expected an error, but none occurred")
					} else {
						t.Logf("OK error: %v", err)
					}
				})
			}
		})
	}
}

func TestPrefixInformationUnmarshalPrefixLength(t *testing.T) {
	// Assume that unmarshaling ignores any prefix bits longer than the
	// specified length.
	var (
		prefix = netip.MustParseAddr("2001:db8::")
		l      = uint8(16)
		want   = netip.MustParseAddr("2001::")
	)

	bs := [][]byte{
		// Option type and length.
		{0x03, 0x04},
		// Prefix Length, shorter than the prefix itself, so the prefix
		// should be cut off.
		{l},
		// Flags, O and A set.
		{0xc0},
		// Valid lifetime.
		{0x00, 0x00, 0x02, 0x58},
		// Preferred lifetime.
		{0x00, 0x00, 0x04, 0xb0},
		// Reserved.
		{0x00, 0x00, 0x00, 0x00},
		// Prefix.
		prefix.AsSlice(),
	}

	pi := new(PrefixInformation)
	if err := pi.unmarshal(ndptest.Merge(bs)); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Assume that unmarshaling ignores any prefix bits longer than the
	// specified length.
	if diff := cmp.Diff(want, pi.Prefix, cmp.Comparer(addrEqual)); diff != "" {
		t.Fatalf("unexpected prefix (-want +got):\n%s", diff)
	}
}

func TestRouteInformationUnmarshalPrefixLength(t *testing.T) {
	// This route prefix easily fits in 2 bytes, but this test will also verify
	// it can be decoded from 3 bytes due to device behaviors seen in the wild.
	var (
		prefix       = netip.MustParseAddr("2001:db8::")
		mask   uint8 = 64
	)

	tests := []struct {
		name   string
		length uint8
		idx    int
	}{
		{
			name:   "length 2",
			length: 2,
			idx:    net.IPv6len / 2,
		},
		{
			name:   "length 3",
			length: 3,
			idx:    net.IPv6len,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := [][]byte{
				// Option type and length. Note that a /64 would normally
				// fit in length 2, but this option was received with padding
				// resulting in length 3.
				{24, tt.length},
				// Prefix length.
				{mask},
				// Preference.
				{0x00},
				// Route lifetime.
				ndptest.Zero(4),
				// Prefix, possibly in a shortened form.
				prefix.AsSlice()[:tt.idx],
			}

			ri := new(RouteInformation)
			if err := ri.unmarshal(ndptest.Merge(bs)); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			want := &RouteInformation{
				PrefixLength: mask,
				Prefix:       prefix,
			}

			if diff := cmp.Diff(want, ri, cmp.Comparer(addrEqual)); diff != "" {
				t.Fatalf("unexpected route information (-want +got):\n%s", diff)
			}
		})
	}
}

func llaTests() []optionSub {
	return []optionSub{
		{
			name: "bad, invalid direction",
			os: []Option{
				&LinkLayerAddress{
					Direction: 10,
				},
			},
		},
		{
			name: "bad, invalid address",
			os: []Option{
				&LinkLayerAddress{
					Direction: Source,
					Addr:      net.HardwareAddr{0xde, 0xad, 0xbe, 0xef},
				},
			},
		},
		{
			name: "ok, source",
			os: []Option{
				&LinkLayerAddress{
					Direction: Source,
					Addr:      ndptest.MAC,
				},
			},
			bs: [][]byte{
				{0x01, 0x01},
				ndptest.MAC,
			},
			ok: true,
		},
		{
			name: "ok, target",
			os: []Option{
				&LinkLayerAddress{
					Direction: Target,
					Addr:      ndptest.MAC,
				},
			},
			bs: [][]byte{
				{0x02, 0x01},
				ndptest.MAC,
			},
			ok: true,
		},
	}
}

func piTests() []optionSub {
	return []optionSub{
		{
			name: "bad, prefix length",
			os: []Option{
				&PrefixInformation{
					// Host IP specified.
					PrefixLength: 64,
					Prefix:       ndptest.IP,
				},
			},
		},
		{
			name: "ok",
			os: []Option{
				&PrefixInformation{
					// Prefix IP specified.
					PrefixLength:                   32,
					OnLink:                         true,
					AutonomousAddressConfiguration: true,
					ValidLifetime:                  Infinity,
					PreferredLifetime:              20 * time.Minute,
					Prefix:                         ndptest.Prefix,
				},
			},
			bs: [][]byte{
				// Option type and length.
				{0x03, 0x04},
				// Prefix Length.
				{32},
				// Flags, O and A set.
				{0xc0},
				// Valid lifetime.
				{0xff, 0xff, 0xff, 0xff},
				// Preferred lifetime.
				{0x00, 0x00, 0x04, 0xb0},
				// Reserved.
				{0x00, 0x00, 0x00, 0x00},
				// Prefix.
				ndptest.Prefix.AsSlice(),
			},
			ok: true,
		},
	}
}

func riTests() []optionSub {
	return []optionSub{
		{
			name: "bad, prefix length",
			os: []Option{
				&RouteInformation{
					// Host IP specified.
					PrefixLength: 64,
					Prefix:       ndptest.IP,
				},
			},
		},
		{
			name: "bad, prefix invalid",
			os: []Option{
				&RouteInformation{
					// Host IP specified.
					PrefixLength: 255,
				},
			},
		},
		{
			name: "ok /0",
			os: []Option{
				&RouteInformation{
					PrefixLength:  0,
					Preference:    High,
					RouteLifetime: Infinity,
					Prefix:        netip.IPv6Unspecified(),
				},
			},
			bs: [][]byte{
				// Option type and length.
				{24, 0x01},
				// Prefix length.
				{0},
				// Preference.
				{0x08},
				// Route lifetime.
				{0xff, 0xff, 0xff, 0xff},
			},
			ok: true,
		},
		{
			name: "ok /64",
			os: []Option{
				&RouteInformation{
					PrefixLength:  64,
					Preference:    Low,
					RouteLifetime: 1 * time.Second,
					Prefix:        ndptest.Prefix,
				},
			},
			bs: [][]byte{
				// Option type and length.
				{24, 0x02},
				// Prefix length.
				{64},
				// Preference.
				{0x18},
				// Route lifetime.
				{0x00, 0x00, 0x00, 0x01},
				// Prefix, second half omitted due to /64 length.
				{0x20, 0x1, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00},
			},
			ok: true,
		},
		{
			name: "ok /96",
			os: []Option{
				&RouteInformation{
					PrefixLength:  96,
					Preference:    Medium,
					RouteLifetime: 255 * time.Second,
					Prefix:        ndptest.Prefix,
				},
			},
			bs: [][]byte{
				// Option type and length.
				{24, 0x03},
				// Prefix length.
				{96},
				// Preference.
				{0x00},
				// Route lifetime.
				{0x00, 0x00, 0x00, 0xff},
				// Prefix, full size due to /96 length.
				{0x20, 0x1, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00},
				ndptest.Zero(8),
			},
			ok: true,
		},
	}
}

func roTests() []optionSub {
	return []optionSub{
		{
			name: "bad, length",
			os: []Option{
				&RawOption{
					Type:   1,
					Length: 1,
					Value:  ndptest.Zero(7),
				},
			},
		},
		{
			name: "ok",
			os: []Option{
				&RawOption{
					Type:   10,
					Length: 2,
					Value:  ndptest.Zero(14),
				},
			},
			bs: [][]byte{
				{0x0a, 0x02},
				ndptest.Zero(14),
			},
			ok: true,
		},
	}
}

func rdnssTests() []optionSub {
	var (
		first  = netip.MustParseAddr("2001:db8::1")
		second = netip.MustParseAddr("2001:db8::2")
	)

	return []optionSub{
		{
			name: "bad, no servers",
			os: []Option{
				&RecursiveDNSServer{
					Lifetime: 1 * time.Second,
				},
			},
		},
		{
			name: "ok, one server",
			os: []Option{
				&RecursiveDNSServer{
					Lifetime: 1 * time.Hour,
					Servers:  []netip.Addr{first},
				},
			},
			bs: [][]byte{
				{25, 3},
				{0x00, 0x00},
				{0x00, 0x00, 0x0e, 0x10},
				first.AsSlice(),
			},
			ok: true,
		},
		{
			name: "ok, two servers",
			os: []Option{
				&RecursiveDNSServer{
					Lifetime: 24 * time.Hour,
					Servers:  []netip.Addr{first, second},
				},
			},
			bs: [][]byte{
				{25, 5},
				{0x00, 0x00},
				{0x00, 0x01, 0x51, 0x80},
				first.AsSlice(),
				second.AsSlice(),
			},
			ok: true,
		},
	}
}

func dnsslTests() []optionSub {
	return []optionSub{
		{
			name: "bad, no domains",
			os: []Option{
				&DNSSearchList{
					Lifetime: 1 * time.Second,
				},
			},
		},
		{
			name: "ok, one domain",
			os: []Option{
				&DNSSearchList{
					Lifetime:    1 * time.Hour,
					DomainNames: []string{"example.com"},
				},
			},
			bs: [][]byte{
				{31, 3},
				// Reserved.
				{0x00, 0x00},
				// Lifetime.
				{0x00, 0x00, 0x0e, 0x10},
				// Labels.
				{7},
				[]byte("example"),
				{3},
				[]byte("com"),
				{0x00},
				// Padding.
				ndptest.Zero(3),
			},
			ok: true,
		},
		{
			name: "ok, multiple servers",
			os: []Option{
				&DNSSearchList{
					Lifetime: 1 * time.Hour,
					DomainNames: []string{
						"example.com",
						"foo.example.com",
						"bar.foo.example.com",
					},
				},
			},
			bs: [][]byte{
				{31, 8},
				// Reserved.
				{0x00, 0x00},
				// Lifetime.
				{0x00, 0x00, 0x0e, 0x10},
				// Labels.
				{7},
				[]byte("example"),
				{3},
				[]byte("com"),
				{0x00},
				{3},
				[]byte("foo"),
				{7},
				[]byte("example"),
				{3},
				[]byte("com"),
				{0x00},
				{3},
				[]byte("bar"),
				{3},
				[]byte("foo"),
				{7},
				[]byte("example"),
				{3},
				[]byte("com"),
				{0x00},
				// Padding.
				ndptest.Zero(5),
			},
			ok: true,
		},
		{
			name: "ok, punycode domain",
			os: []Option{
				&DNSSearchList{
					Lifetime:    1 * time.Hour,
					DomainNames: []string{"😃.example.com"},
				},
			},
			bs: [][]byte{
				{31, 4},
				// Reserved.
				{0x00, 0x00},
				// Lifetime.
				{0x00, 0x00, 0x0e, 0x10},
				// Labels.
				{8},
				[]byte("xn--h28h"),
				{7},
				[]byte("example"),
				{3},
				[]byte("com"),
				{0x00},
				// Padding.
				ndptest.Zero(2),
			},
			ok: true,
		},
	}
}

func cpTests() []optionSub {
	return []optionSub{
		{
			name: "bad, empty",
			os: []Option{
				NewCaptivePortal(""),
			},
		},
		{
			name: "ok, no padding",
			os: []Option{
				NewCaptivePortal("urn:xx"),
			},
			bs: [][]byte{
				{37, 1},
				// URI.
				{'u', 'r', 'n', ':', 'x', 'x'},
			},
			ok: true,
		},
		{
			name: "ok, padding",
			os: []Option{
				NewCaptivePortal("capport:unrestricted"),
			},
			bs: [][]byte{
				{37, 3},
				// URI.
				{'c', 'a', 'p', 'p', 'o', 'r', 't', ':', 'u', 'n', 'r', 'e', 's', 't', 'r', 'i', 'c', 't', 'e', 'd'},
				// Padding.
				ndptest.Zero(2),
			},
			ok: true,
		},
	}
}
