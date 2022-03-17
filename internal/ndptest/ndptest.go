// Package ndptest provides test functions and types for package ndp.
package ndptest

import (
	"bytes"
	"net"
	"net/netip"
)

// Shared test data for commonly needed data types.
var (
	Prefix = netip.MustParseAddr("2001:db8::")
	IP     = netip.MustParseAddr("2001:db8::1")
	MAC    = net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}
)

// Merge merges a slice of byte slices into a single, contiguous slice.
func Merge(bs [][]byte) []byte {
	var b []byte
	for _, bb := range bs {
		b = append(b, bb...)
	}

	return b
}

// Zero returns a byte slice of size n filled with zeros.
func Zero(n int) []byte {
	return bytes.Repeat([]byte{0x00}, n)
}
