package ndpcmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/mdlayher/ndp"
)

func sendNS(ctx context.Context, c *ndp.Conn, addr net.HardwareAddr, target net.IP) error {
	ll := log.New(os.Stderr, "ndp ns> ", 0)

	ll.Printf("neighbor solicitation:\n    - source link-layer address: %s", addr.String())

	m := &ndp.NeighborSolicitation{
		TargetAddress: target,
		Options: []ndp.Option{
			&ndp.LinkLayerAddress{
				Direction: ndp.Source,
				Addr:      addr,
			},
		},
	}

	for i := 0; ; i++ {
		if err := c.WriteTo(m, nil, target); err != nil {
			return fmt.Errorf("failed to write neighbor solicitation: %v", err)
		}

		na, from, err := receiveNA(c)
		if err == nil {
			fmt.Println()
			printNA(ll, na, from)
			return nil
		}

		// Was the context canceled already?
		select {
		case <-ctx.Done():
			fmt.Println()
			ll.Printf("sent %d neighbor solicitation(s)", i+1)
			return ctx.Err()
		default:
		}

		// Was the error caused by a read timeout, and should the loop continue?
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			fmt.Print(".")
			continue
		}

		return fmt.Errorf("failed to read neighbor advertisement: %v", err)
	}
}

func receiveNA(c *ndp.Conn) (*ndp.NeighborAdvertisement, net.IP, error) {
	if err := c.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		return nil, nil, err
	}

	for {
		msg, _, from, err := c.ReadFrom()
		if err != nil {
			return nil, nil, err
		}

		na, ok := msg.(*ndp.NeighborAdvertisement)
		if !ok {
			continue
		}

		return na, from, nil
	}
}
