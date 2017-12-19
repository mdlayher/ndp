package ndp_test

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/ndp"
)

func TestMarshalParseMessage(t *testing.T) {
	ip := mustIPv6("2001:db8:dead:beef:f00::00d")

	tests := []struct {
		name string
		m    ndp.Message
		b    []byte
	}{
		{
			name: "neighbor advertisement",
			m: &ndp.NeighborAdvertisement{
				Router:        true,
				Solicited:     true,
				Override:      true,
				TargetAddress: ip,
			},
			b: append([]byte{
				136, 0x00, 0x00, 0x00,
				0xe0, 0x00, 0x00, 0x00,
			}, ip...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := ndp.MarshalMessage(tt.m)
			if err != nil {
				t.Fatalf("failed to marshal message: %v", err)
			}

			if diff := cmp.Diff(tt.b, b); diff != "" {
				t.Fatalf("unexpected message bytes (-want +got):\n%s", diff)
			}

			m, err := ndp.ParseMessage(b)
			if err != nil {
				t.Fatalf("failed to unmarshal message: %v", err)
			}

			if diff := cmp.Diff(tt.m, m); diff != "" {
				t.Fatalf("unexpected message (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseMessage(t *testing.T) {
	ip := mustIPv6("::")

	tests := []struct {
		name string
		b    []byte
		m    ndp.Message
		ok   bool
	}{
		{
			name: "bad, short",
			b: []byte{
				255,
			},
		},
		{
			name: "bad, unknown type",
			b: []byte{
				255, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "ok, neighbor advertisement",
			b: append([]byte{
				136, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			}, ip...),
			m:  &ndp.NeighborAdvertisement{},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := ndp.ParseMessage(tt.b)

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

			if typ := reflect.TypeOf(m); typ != reflect.TypeOf(tt.m) {
				t.Fatalf("unexpected message type: %T", typ)
			}
		})
	}
}

func TestNeighborAdvertisementMarshalUnmarshalBinary(t *testing.T) {
	ip := mustIPv6("2001:db8::1")
	bfn := func(b []byte) []byte {
		return append(b, ip...)
	}

	tests := []struct {
		name string
		na   *ndp.NeighborAdvertisement
		b    []byte
		ok   bool
	}{
		{
			name: "bad, malformed IP address",
			na: &ndp.NeighborAdvertisement{
				TargetAddress: net.IP{192, 168, 1, 1, 0, 0},
			},
		},
		{
			name: "bad, IPv4 address",
			na: &ndp.NeighborAdvertisement{
				TargetAddress: net.IPv4(192, 168, 1, 1),
			},
		},
		{
			name: "ok, no flags",
			na: &ndp.NeighborAdvertisement{
				TargetAddress: ip,
			},
			b:  bfn([]byte{0x00, 0x00, 0x00, 0x00}),
			ok: true,
		},
		{
			name: "ok, router",
			na: &ndp.NeighborAdvertisement{
				Router:        true,
				TargetAddress: ip,
			},
			b:  bfn([]byte{0x80, 0x00, 0x00, 0x00}),
			ok: true,
		},
		{
			name: "ok, solicited",
			na: &ndp.NeighborAdvertisement{
				Solicited:     true,
				TargetAddress: ip,
			},
			b:  bfn([]byte{0x40, 0x00, 0x00, 0x00}),
			ok: true,
		},
		{
			name: "ok, override",
			na: &ndp.NeighborAdvertisement{
				Override:      true,
				TargetAddress: ip,
			},
			b:  bfn([]byte{0x20, 0x00, 0x00, 0x00}),
			ok: true,
		},
		{
			name: "ok, all flags",
			na: &ndp.NeighborAdvertisement{
				Router:        true,
				Solicited:     true,
				Override:      true,
				TargetAddress: ip,
			},
			b:  bfn([]byte{0xe0, 0x00, 0x00, 0x00}),
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.na.MarshalBinary()

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

			if diff := cmp.Diff(tt.b, b); diff != "" {
				t.Fatalf("unexpected message bytes (-want +got):\n%s", diff)
			}

			na := new(ndp.NeighborAdvertisement)
			if err := na.UnmarshalBinary(b); err != nil {
				t.Fatalf("failed to unmarshal binary: %v", err)
			}

			if diff := cmp.Diff(tt.na, na); diff != "" {
				t.Fatalf("unexpected neighbor advertisement (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNeighborAdvertisementUnmarshalBinary(t *testing.T) {
	ip := mustIPv6("2001:db8:dead:beef:f00::00d")

	tests := []struct {
		name string
		b    []byte
		na   *ndp.NeighborAdvertisement
		ok   bool
	}{
		{
			name: "bad, short",
			b:    ip,
		},
		{
			name: "ok",
			b: append([]byte{
				0xe0, 0x00, 0x00, 0x00,
			}, ip...),
			na: &ndp.NeighborAdvertisement{
				Router:        true,
				Solicited:     true,
				Override:      true,
				TargetAddress: ip,
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			na := new(ndp.NeighborAdvertisement)
			err := na.UnmarshalBinary(tt.b)

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

			if diff := cmp.Diff(tt.na, na); diff != "" {
				t.Fatalf("unexpected neighbor advertisement (-want +got):\n%s", diff)
			}
		})
	}
}

func mustIPv6(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil || ip.To4() != nil {
		panic(fmt.Sprintf("invalid IPv6 address: %q", s))
	}

	return ip
}
