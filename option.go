package ndp

import (
	"encoding"
	"fmt"
	"io"
	"net"
)

const (
	// Length of a link-layer address for Ethernet networks.
	ethAddrLen = 6

	// The assumed NDP option length (in units of 8 bytes) for a source or
	// target link layer address option for Ethernet networks.
	llaOptLen = 1

	// Byte length values required for each type of valid Option.
	llaByteLen = 8
)

// A Direction specifies the direction of an Option as a source or target.
type Direction int

// Possible Direction values.
const (
	Source Direction = 1
	Target Direction = 2
)

// An Option is a Neighbor Discovery Protocol option.
type Option interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	code() uint8
}

var _ Option = &LinkLayerAddress{}

// A LinkLayerAddress is a Source or Target Link-Layer Address option, as
// described in RFC 4861, Section 4.6.1.
type LinkLayerAddress struct {
	Direction Direction
	Addr      net.HardwareAddr
}

// TODO(mdlayher): deal with non-ethernet links and variable option length?

func (lla *LinkLayerAddress) code() byte { return byte(lla.Direction) }

// MarshalBinary implements Option.
func (lla *LinkLayerAddress) MarshalBinary() ([]byte, error) {
	if d := lla.Direction; d != Source && d != Target {
		return nil, fmt.Errorf("ndp: invalid link-layer address direction: %d", d)
	}

	if len(lla.Addr) != ethAddrLen {
		return nil, fmt.Errorf("ndp: invalid link-layer address: %q", lla.Addr.String())
	}

	b := make([]byte, llaByteLen)
	b[0] = lla.code()
	b[1] = llaOptLen
	copy(b[2:], lla.Addr)

	return b, nil
}

// UnmarshalBinary implements Option.
func (lla *LinkLayerAddress) UnmarshalBinary(b []byte) error {
	if len(b) < llaByteLen {
		return io.ErrUnexpectedEOF
	}

	d := Direction(b[0])
	if d != Source && d != Target {
		return fmt.Errorf("ndp: invalid link-layer address direction: %d", d)
	}

	if l := b[1]; l != 1 {
		return fmt.Errorf("ndp: unexpected link-layer address option length: %d", l)
	}

	*lla = LinkLayerAddress{
		Direction: d,
		Addr:      make(net.HardwareAddr, 6),
	}

	copy(lla.Addr, b[2:])

	return nil
}
