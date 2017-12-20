package ndp

import (
	"encoding"
	"fmt"
	"io"
	"net"
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

	if len(lla.Addr) != 6 {
		return nil, fmt.Errorf("ndp: invalid link-layer address: %q", lla.Addr.String())
	}

	b := make([]byte, 8)
	b[0] = lla.code()
	b[1] = 1
	copy(b[2:], lla.Addr)

	return b, nil
}

// UnmarshalBinary implements Option.
func (lla *LinkLayerAddress) UnmarshalBinary(b []byte) error {
	if len(b) < 8 {
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
