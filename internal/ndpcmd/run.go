// Package ndpcmd provides the commands for the ndp utility.
package ndpcmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"

	"github.com/mdlayher/ndp"
)

var errTargetOp = errors.New("flag '-t' is only valid for neighbor solicitation operation")

// Run runs the ndp utility.
func Run(
	ctx context.Context,
	c *ndp.Conn,
	ifi *net.Interface,
	op string,
	target netip.Addr,
) error {
	if op != "ns" && target.IsValid() {
		return errTargetOp
	}

	switch op {
	// listen is the default when no op is specified.
	case "listen", "":
		return listen(ctx, c)
	case "ns":
		return sendNS(ctx, c, ifi.HardwareAddr, target)
	case "rs":
		return sendRS(ctx, c, ifi.HardwareAddr)
	default:
		return fmt.Errorf("unrecognized operation: %q", op)
	}
}

func listen(ctx context.Context, c *ndp.Conn) error {
	ll := log.New(os.Stderr, "ndp listen> ", 0)
	ll.Println("listening for messages")

	// Also listen for router solicitations from other hosts, even though we
	// will never reply to them.
	if err := c.JoinGroup(netip.MustParseAddr("ff02::2")); err != nil {
		return err
	}

	// No filtering, print all messages.
	if err := receiveLoop(ctx, c, ll, nil, nil); err != nil {
		return fmt.Errorf("failed to read message: %v", err)
	}

	return nil
}

func sendNS(ctx context.Context, c *ndp.Conn, addr net.HardwareAddr, target netip.Addr) error {
	ll := log.New(os.Stderr, "ndp ns> ", 0)

	ll.Printf("neighbor solicitation:\n    - source link-layer address: %s", addr.String())

	// Always multicast the message to the target's solicited-node multicast
	// group as if we have no knowledge of its MAC address.
	snm, err := ndp.SolicitedNodeMulticast(target)
	if err != nil {
		return fmt.Errorf("failed to determine solicited-node multicast address: %v", err)
	}

	m := &ndp.NeighborSolicitation{
		TargetAddress: target,
		Options: []ndp.Option{
			&ndp.LinkLayerAddress{
				Direction: ndp.Source,
				Addr:      addr,
			},
		},
	}

	// Expect neighbor advertisement messages with the correct target address.
	check := func(m ndp.Message) bool {
		na, ok := m.(*ndp.NeighborAdvertisement)
		if !ok {
			return false
		}

		return na.TargetAddress == target
	}

	if err := sendReceiveLoop(ctx, c, ll, m, snm, check); err != nil {
		if err == context.Canceled {
			return err
		}

		return fmt.Errorf("failed to send neighbor solicitation: %v", err)
	}

	return nil
}

func sendRS(ctx context.Context, c *ndp.Conn, addr net.HardwareAddr) error {
	ll := log.New(os.Stderr, "ndp rs> ", 0)

	// Non-Ethernet interfaces (such as PPPoE) may not have a MAC address, so
	// optionally set the source LLA option if addr is set.
	m := &ndp.RouterSolicitation{}
	msg := "router solicitation:"
	if addr != nil {
		msg += fmt.Sprintf("\n  - source link-layer address: %s", addr.String())

		m.Options = append(m.Options, &ndp.LinkLayerAddress{
			Direction: ndp.Source,
			Addr:      addr,
		})
	}

	ll.Println(msg)

	// Expect any router advertisement message.
	check := func(m ndp.Message) bool {
		_, ok := m.(*ndp.RouterAdvertisement)
		return ok
	}

	if err := sendReceiveLoop(ctx, c, ll, m, netip.MustParseAddr("ff02::2"), check); err != nil {
		if err == context.Canceled {
			return err
		}

		return fmt.Errorf("failed to send router solicitation: %v", err)
	}

	return nil
}
