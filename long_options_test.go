package ndp_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/netip"
	"strings"
	"testing"

	"github.com/mdlayher/ndp"
)

func TestRawOptionLengthUnits(t *testing.T) {
	for units := 1; units <= 255; units++ {
		t.Run(fmt.Sprint(units), func(t *testing.T) {
			// Construct the wire packet independently of the marshaler. Retain a
			// following option to check that decoding consumes exactly this length.
			option := make([]byte, units*8)
			option[0], option[1] = 253, byte(units)
			for i := 2; i < len(option); i++ {
				option[i] = byte(i)
			}
			packet := append([]byte{134, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, option...)
			packet = append(packet, 5, 1, 0, 0, 0, 0, 5, 220)
			message, err := ndp.ParseMessage(packet)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			ra := message.(*ndp.RouterAdvertisement)
			if len(ra.Options) != 2 {
				t.Fatalf("options: %d", len(ra.Options))
			}
			raw := ra.Options[0].(*ndp.RawOption)
			if raw.Length != byte(units) || !bytes.Equal(raw.Value, option[2:]) {
				t.Fatal("raw value changed")
			}
			if mtu := ra.Options[1].(*ndp.MTU).MTU; mtu != 1500 {
				t.Fatalf("MTU: %d", mtu)
			}
			got, err := ndp.MarshalMessage(message)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !bytes.Equal(got, packet) {
				t.Fatal("packet changed")
			}
			for _, value := range [][]byte{raw.Value[:len(raw.Value)-1], append(append([]byte{}, raw.Value...), 0)} {
				if _, err := ndp.MarshalMessage(&ndp.RouterAdvertisement{Options: []ndp.Option{&ndp.RawOption{Type: 253, Length: byte(units), Value: value}}}); err == nil {
					t.Fatal("accepted mismatched raw length")
				}
			}
			// A declared option which is one byte short must still be rejected.
			if _, err := ndp.ParseMessage(packet[:16+len(option)-1]); err == nil {
				t.Fatal("accepted truncated option")
			}
		})
	}
}

func TestCaptivePortalLengthBoundary(t *testing.T) {
	for length := 240; length <= 255; length++ {
		t.Run(fmt.Sprint(length), func(t *testing.T) {
			uri := "https://example.com/" + strings.Repeat("x", length-len("https://example.com/"))
			cp, err := ndp.NewCaptivePortal(uri)
			if err != nil {
				t.Fatal(err)
			}
			packet, err := ndp.MarshalMessage(&ndp.RouterAdvertisement{Options: []ndp.Option{cp, ndp.NewMTU(1500)}})
			if err != nil {
				t.Fatal(err)
			}
			size := ((length + 2 + 7) / 8) * 8
			if len(packet) != 16+size+8 || packet[16] != 37 || int(packet[17]) != size/8 {
				t.Fatal("incorrect option size")
			}
			if string(packet[18:18+length]) != uri {
				t.Fatal("URI changed")
			}
			if !bytes.Equal(packet[18+length:16+size], make([]byte, size-length-2)) {
				t.Fatal("nonzero padding")
			}
			message, err := ndp.ParseMessage(packet)
			if err != nil {
				t.Fatal(err)
			}
			options := message.(*ndp.RouterAdvertisement).Options
			if len(options) != 2 || options[0].(*ndp.CaptivePortal).URI != uri || options[1].(*ndp.MTU).MTU != 1500 {
				t.Fatal("parsed options changed")
			}
		})
	}
	if _, err := ndp.NewCaptivePortal("https://example.com/" + strings.Repeat("x", 256-len("https://example.com/"))); err == nil {
		t.Fatal("constructor accepted oversized URI")
	}
	for _, length := range []int{0, 2039, 2040, 4094} {
		if _, err := ndp.MarshalMessage(&ndp.RouterAdvertisement{Options: []ndp.Option{&ndp.CaptivePortal{URI: strings.Repeat("x", length)}}}); err == nil {
			t.Fatalf("accepted invalid URI length %d", length)
		}
	}
}

func TestLongTypedOptions(t *testing.T) {
	servers := make([]netip.Addr, 16)
	for i := range servers {
		servers[i] = netip.MustParseAddr(fmt.Sprintf("2001:db8::%x", i+1))
	}
	for _, option := range []ndp.Option{
		&ndp.RecursiveDNSServer{Servers: servers},
		&ndp.DNSSearchList{DomainNames: []string{strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + ".example", strings.Repeat("c", 63) + "." + strings.Repeat("d", 63) + ".example"}},
		&ndp.CaptivePortal{URI: "https://example.com/" + strings.Repeat("x", 2038-len("https://example.com/"))},
		&ndp.RAFlagsExtension{Flags: append([]byte{1}, make([]byte, 253)...)},
	} {
		t.Run(fmt.Sprint(option.Code()), func(t *testing.T) {
			packet, err := ndp.MarshalMessage(&ndp.RouterAdvertisement{Options: []ndp.Option{option, ndp.NewMTU(1500)}})
			if err != nil {
				t.Fatal(err)
			}
			if packet[17] < 32 {
				t.Fatal("test did not produce long option")
			}
			message, err := ndp.ParseMessage(packet)
			if err != nil {
				t.Fatal(err)
			}
			again, err := ndp.MarshalMessage(message)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(packet, again) {
				t.Fatal("typed option changed")
			}
		})
	}
	// Long nonces arrive through parsing; their state is deliberately private.
	for _, code := range []byte{14, 26} {
		packet := make([]byte, 16+256+8)
		packet[0], packet[16], packet[17], packet[18] = 134, code, 32, 1
		copy(packet[16+256:], []byte{5, 1, 0, 0, 0, 0, 5, 220})
		binary.BigEndian.PutUint16(packet[6:8], 30)
		message, err := ndp.ParseMessage(packet)
		if err != nil {
			t.Fatalf("parse code %d: %v", code, err)
		}
		again, err := ndp.MarshalMessage(message)
		if err != nil {
			t.Fatalf("marshal code %d: %v", code, err)
		}
		if !bytes.Equal(packet, again) {
			t.Fatalf("code %d changed", code)
		}
	}
}
