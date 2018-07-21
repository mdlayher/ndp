// Package ndpcmd provides the commands for the ndp utility.
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

// Run runs the ndp utility.
func Run(ctx context.Context, c *ndp.Conn, ifi *net.Interface, op string, target net.IP) error {
	switch op {
	// listen is the default when no op is specified..
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
	if err := c.JoinGroup(net.IPv6linklocalallrouters); err != nil {
		return err
	}

	var recv int
	for {
		if err := c.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			return err
		}

		m, _, from, err := c.ReadFrom()
		if err == nil {
			recv++
			printMessage(ll, m, from)
			continue
		}

		// Was the context canceled already?
		select {
		case <-ctx.Done():
			ll.Printf("received %d message(s)", recv)
			return ctx.Err()
		default:
		}

		// Was the error caused by a read timeout, and should the loop continue?
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			continue
		}

		return fmt.Errorf("failed to read message: %v", err)
	}
}
