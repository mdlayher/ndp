package ndp

import (
	"fmt"
	"net"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

// A Conn is a Neighbor Discovery Protocol connection.
type Conn struct {
	pc *ipv6.PacketConn
	cm *ipv6.ControlMessage

	allNodes *net.IPAddr
}

// Dial dials a NDP connection using the specified interface.  It returns
// a Conn and the link-local IPv6 address of the interface.
func Dial(ifi *net.Interface) (*Conn, net.IP, error) {
	llAddr, err := linkLocalAddr(ifi)
	if err != nil {
		return nil, nil, err
	}

	ic, err := icmp.ListenPacket("ip6:ipv6-icmp", llAddr.String())
	if err != nil {
		return nil, nil, err
	}

	// Join the "all nodes" multicast group for this interface.
	allNodes := &net.IPAddr{
		IP:   net.IPv6linklocalallnodes,
		Zone: ifi.Name,
	}

	pc := ic.IPv6PacketConn()
	if err := pc.JoinGroup(ifi, allNodes); err != nil {
		return nil, nil, err
	}

	// Calculate and place ICMPv6 checksum at correct offset in all messages.
	if err := pc.SetChecksum(true, 2); err != nil {
		return nil, nil, err
	}

	c := &Conn{
		pc: pc,

		// The default control message used when none is specified.
		cm: &ipv6.ControlMessage{
			// Note that hop limit is always 255 in NDP.
			HopLimit: 255,
			Src:      llAddr.IP,
			IfIndex:  ifi.Index,
		},

		allNodes: allNodes,
	}

	return c, llAddr.IP, nil
}

// Close closes the Conn's underlying connection.
func (c *Conn) Close() error {
	return c.pc.Close()
}

// WriteTo writes a Message to the Conn, with an optional control message and
// destination network address.
//
// If cm is nil, a default control message will be sent.  If dst is nil, the
// message will be sent to the IPv6 link-local all nodes address for the Conn.
func (c *Conn) WriteTo(m Message, cm *ipv6.ControlMessage, dst net.IP) error {
	b, err := MarshalMessage(m)
	if err != nil {
		return err
	}

	// Set reasonable defaults if control message or destination are nil.
	if cm == nil {
		cm = c.cm
	}

	addr := &net.IPAddr{IP: dst}
	if dst == nil {
		addr = c.allNodes
	}

	_, err = c.pc.WriteTo(b, cm, addr)
	return err
}

// linkLocalPrefix is the IPv6 link-local prefix fe80::/10.
var linkLocalPrefix = &net.IPNet{
	IP:   net.ParseIP("fe80::"),
	Mask: net.CIDRMask(10, 128),
}

// linkLocalAddr searches for a valid IPv6 link-local address for the specified
// interface.
func linkLocalAddr(ifi *net.Interface) (*net.IPAddr, error) {
	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, err
	}

	for _, a := range addrs {
		ipn, ok := a.(*net.IPNet)
		if !ok {
			continue
		}

		if err := checkIPv6(ipn.IP); err != nil {
			continue
		}

		if !linkLocalPrefix.Contains(ipn.IP) {
			continue
		}

		return &net.IPAddr{
			IP:   ipn.IP,
			Zone: ifi.Name,
		}, nil
	}

	return nil, fmt.Errorf("ndp: no link local address for interface %q", ifi.Name)
}
