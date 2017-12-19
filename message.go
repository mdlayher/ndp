package ndp

import (
	"encoding"
	"fmt"
	"io"
	"net"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

// A Message is a Neighbor Discovery Protocol message.
type Message interface {
	Type() ipv6.ICMPType
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// MarshalMessage marshals a Message into its binary form and prepends an
// ICMPv6 message with the correct type.
//
// It is assumed that the operating system or caller will calculate and place
// the ICMPv6 checksum in the result.
func MarshalMessage(m Message) ([]byte, error) {
	mb, err := m.MarshalBinary()
	if err != nil {
		return nil, err
	}

	im := icmp.Message{
		Type: m.Type(),
		// Always zero.
		Code: 0,
		// Calculated by caller or OS.
		Checksum: 0,
		// TODO(mdlayher): is this the correct MessageBody implementation?
		Body: &icmp.DefaultMessageBody{
			Data: mb,
		},
	}

	// Pseudo-header always nil so checksum is calculated by caller or OS.
	return im.Marshal(nil)
}

// ParseMessage parses a Message from its binary form after determining its
// type from a leading ICMPv6 message.
func ParseMessage(b []byte) (Message, error) {
	if len(b) < 4 {
		return nil, io.ErrUnexpectedEOF
	}

	// TODO(mdlayher): verify checksum?

	var m Message
	switch t := ipv6.ICMPType(b[0]); t {
	case ipv6.ICMPTypeNeighborAdvertisement:
		m = new(NeighborAdvertisement)
	default:
		return nil, fmt.Errorf("ndp: unrecognized ICMPv6 type: %d", t)
	}

	if err := m.UnmarshalBinary(b[4:]); err != nil {
		return nil, err
	}

	return m, nil
}

var _ Message = &NeighborAdvertisement{}

// A NeighborAdvertisement is a Neighbor Advertisement message as
// described in RFC 4861, Section 4.4.
type NeighborAdvertisement struct {
	Router        bool
	Solicited     bool
	Override      bool
	TargetAddress net.IP

	// TODO(mdlayher): options type.
}

// Type implements Message.
func (na *NeighborAdvertisement) Type() ipv6.ICMPType {
	return ipv6.ICMPTypeNeighborAdvertisement
}

// MarshalBinary implements Message.
func (na *NeighborAdvertisement) MarshalBinary() ([]byte, error) {
	if err := checkIPv6(na.TargetAddress); err != nil {
		return nil, err
	}

	b := make([]byte, 20)

	if na.Router {
		b[0] |= (1 << 7)
	}
	if na.Solicited {
		b[0] |= (1 << 6)
	}
	if na.Override {
		b[0] |= (1 << 5)
	}

	copy(b[4:], na.TargetAddress)

	return b, nil
}

// UnmarshalBinary implements Message.
func (na *NeighborAdvertisement) UnmarshalBinary(b []byte) error {
	if len(b) < 20 {
		return io.ErrUnexpectedEOF
	}

	addr := b[4 : 4+net.IPv6len]
	if err := checkIPv6(addr); err != nil {
		return err
	}

	*na = NeighborAdvertisement{
		Router:    (b[0] & 0x80) != 0,
		Solicited: (b[0] & 0x40) != 0,
		Override:  (b[0] & 0x20) != 0,

		TargetAddress: make(net.IP, net.IPv6len),
	}

	copy(na.TargetAddress, addr)

	return nil
}

// checkIPv6 verifies that ip is an IPv6 address.
func checkIPv6(ip net.IP) error {
	if ip.To16() == nil || ip.To4() != nil {
		return fmt.Errorf("ndp: invalid IPv6 address: %q", ip.String())
	}

	return nil
}
