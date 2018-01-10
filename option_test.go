package ndp_test

import (
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/ndp"
)

func TestLinkLayerAddressMarshalUnmarshalBinary(t *testing.T) {
	addr := net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}

	tests := []struct {
		name string
		lla  *ndp.LinkLayerAddress
		bs   [][]byte
		ok   bool
	}{
		{
			name: "bad, invalid direction",
			lla: &ndp.LinkLayerAddress{
				Direction: 10,
			},
		},
		{
			name: "bad, invalid address",
			lla: &ndp.LinkLayerAddress{
				Direction: ndp.Source,
				Addr:      net.HardwareAddr{0xde, 0xad, 0xbe, 0xef},
			},
		},
		{
			name: "ok, source",
			lla: &ndp.LinkLayerAddress{
				Direction: ndp.Source,
				Addr:      addr,
			},
			bs: [][]byte{
				{0x01, 0x01},
				addr,
			},
			ok: true,
		},
		{
			name: "ok, target",
			lla: &ndp.LinkLayerAddress{
				Direction: ndp.Target,
				Addr:      addr,
			},
			bs: [][]byte{
				{0x02, 0x01},
				addr,
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.lla.MarshalBinary()

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

			ttb := merge(tt.bs)
			if diff := cmp.Diff(ttb, b); diff != "" {
				t.Fatalf("unexpected Option bytes (-want +got):\n%s", diff)
			}

			lla := new(ndp.LinkLayerAddress)
			if err := lla.UnmarshalBinary(b); err != nil {
				t.Fatalf("failed to unmarshal binary: %v", err)
			}

			if diff := cmp.Diff(tt.lla, lla); diff != "" {
				t.Fatalf("unexpected link-layer address (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLinkLayerAddressUnmarshalBinary(t *testing.T) {
	addr := net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}

	tests := []struct {
		name string
		bs   [][]byte
		lla  *ndp.LinkLayerAddress
		ok   bool
	}{
		{
			name: "bad, short",
			bs:   [][]byte{{0x01, 0x01, 0xff}},
		},
		{
			name: "bad, invalid direction",
			bs: [][]byte{
				{0x10, 0x01},
				addr,
			},
		},
		{
			name: "bad, long",
			bs: [][]byte{
				{0x01, 0x02},
				zero(16),
			},
		},
		{
			name: "ok, source",
			bs: [][]byte{
				{0x01, 0x01},
				addr,
			},
			lla: &ndp.LinkLayerAddress{
				Direction: ndp.Source,
				Addr:      addr,
			},
			ok: true,
		},
		{
			name: "ok, target",
			bs: [][]byte{
				{0x02, 0x01},
				addr,
			},
			lla: &ndp.LinkLayerAddress{
				Direction: ndp.Target,
				Addr:      addr,
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lla := new(ndp.LinkLayerAddress)
			err := lla.UnmarshalBinary(merge(tt.bs))

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

			if diff := cmp.Diff(tt.lla, lla); diff != "" {
				t.Fatalf("unexpected link-layer address (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRawOptionMarshalUnmarshalBinary(t *testing.T) {
	tests := []struct {
		name string
		ro   *ndp.RawOption
		bs   [][]byte
		ok   bool
	}{
		{
			name: "bad, length",
			ro: &ndp.RawOption{
				Type:   1,
				Length: 1,
				Value:  zero(7),
			},
		},
		{
			name: "ok",
			ro: &ndp.RawOption{
				Type:   10,
				Length: 2,
				Value:  zero(14),
			},
			bs: [][]byte{
				{0x0a, 0x02},
				zero(14),
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.ro.MarshalBinary()

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

			ttb := merge(tt.bs)
			if diff := cmp.Diff(ttb, b); diff != "" {
				t.Fatalf("unexpected Option bytes (-want +got):\n%s", diff)
			}

			ro := new(ndp.RawOption)
			if err := ro.UnmarshalBinary(b); err != nil {
				t.Fatalf("failed to unmarshal binary: %v", err)
			}

			if diff := cmp.Diff(tt.ro, ro); diff != "" {
				t.Fatalf("unexpected raw option (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRawOptionUnmarshalBinary(t *testing.T) {
	tests := []struct {
		name string
		bs   [][]byte
		ro   *ndp.RawOption
		ok   bool
	}{
		{
			name: "bad, short",
			bs:   [][]byte{{0x01}},
		},
		{
			name: "bad, misleading length",
			bs:   [][]byte{{0x10, 0x10}},
		},
		{
			name: "ok",
			bs: [][]byte{
				{0xff, 0x02},
				zero(14),
			},
			ro: &ndp.RawOption{
				Type:   255,
				Length: 2,
				Value:  zero(14),
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ro := new(ndp.RawOption)
			err := ro.UnmarshalBinary(merge(tt.bs))

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

			if diff := cmp.Diff(tt.ro, ro); diff != "" {
				t.Fatalf("unexpected raw option (-want +got):\n%s", diff)
			}
		})
	}
}
