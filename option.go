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

	// Type values for each type of valid Option.
	optSourceLLA = 1
	optTargetLLA = 2
)

// A Direction specifies the direction of a LinkLayerAddress Option as a source
// or target.
type Direction int

// Possible Direction values.
const (
	Source Direction = optSourceLLA
	Target Direction = optTargetLLA
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

	raw := &RawOption{
		Type:   lla.code(),
		Length: llaOptLen,
		Value:  lla.Addr,
	}

	return raw.MarshalBinary()
}

// UnmarshalBinary implements Option.
func (lla *LinkLayerAddress) UnmarshalBinary(b []byte) error {
	raw := new(RawOption)
	if err := raw.UnmarshalBinary(b); err != nil {
		return err
	}

	d := Direction(raw.Type)
	if d != Source && d != Target {
		return fmt.Errorf("ndp: invalid link-layer address direction: %d", d)
	}

	if l := raw.Length; l != llaOptLen {
		return fmt.Errorf("ndp: unexpected link-layer address option length: %d", l)
	}

	*lla = LinkLayerAddress{
		Direction: d,
		Addr:      net.HardwareAddr(raw.Value),
	}

	return nil
}

var _ Option = &RawOption{}

// A RawOption is an Option in its raw and unprocessed format.  Options which
// are not recognized by this package can be represented using a RawOption.
type RawOption struct {
	Type   uint8
	Length uint8
	Value  []byte
}

func (u *RawOption) code() byte { return u.Type }

// MarshalBinary implements Option.
func (u *RawOption) MarshalBinary() ([]byte, error) {
	// Length specified in units of 8 bytes, and the caller must provide
	// an accurate length.
	l := int(u.Length * 8)
	if 1+1+len(u.Value) != l {
		return nil, io.ErrUnexpectedEOF
	}

	b := make([]byte, u.Length*8)
	b[0] = u.Type
	b[1] = u.Length

	copy(b[2:], u.Value)

	return b, nil
}

// UnmarshalBinary implements Option.
func (u *RawOption) UnmarshalBinary(b []byte) error {
	if len(b) < 2 {
		return io.ErrUnexpectedEOF
	}

	u.Type = b[0]
	u.Length = b[1]
	// Exclude type and length fields from value's length.
	l := int(u.Length*8) - 2

	if l > len(b[2:]) {
		return io.ErrUnexpectedEOF
	}

	u.Value = make([]byte, l)
	copy(u.Value, b[2:])

	return nil
}

// marshalOptions marshals a slice of Options into a single byte slice.
func marshalOptions(options []Option) ([]byte, error) {
	var b []byte
	for _, o := range options {
		ob, err := o.MarshalBinary()
		if err != nil {
			return nil, err
		}

		b = append(b, ob...)
	}

	return b, nil
}

// parseOptions parses a slice of Options from a byte slice.
func parseOptions(b []byte) ([]Option, error) {
	var options []Option
	for i := 0; len(b[i:]) != 0; {
		// Two bytes: option type and option length.
		if len(b[i:]) < 2 {
			return nil, io.ErrUnexpectedEOF
		}

		// Type processed as-is, but length is stored in units of 8 bytes,
		// so expand it to the actual byte length.
		t := b[i]
		l := int(b[i+1]) * 8

		// Infer the option from its type value and use it for unmarshaling.
		var o Option
		switch t {
		case optSourceLLA, optTargetLLA:
			o = new(LinkLayerAddress)
		default:
			o = new(RawOption)
		}

		// Unmarshal at the current offset, up to the expected length.
		if err := o.UnmarshalBinary(b[i : i+l]); err != nil {
			return nil, err
		}

		// Verify that we won't advance beyond the end of the byte slice, and
		// Advance to the next option's type field.
		if l > len(b[i:]) {
			return nil, io.ErrUnexpectedEOF
		}
		i += l

		options = append(options, o)
	}

	return options, nil
}
