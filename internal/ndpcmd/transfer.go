package ndpcmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"time"

	"github.com/mdlayher/ndp"
)

func sendReceiveLoop(
	ctx context.Context,
	c *ndp.Conn,
	ll *log.Logger,
	m ndp.Message,
	dst netip.Addr,
	check func(m ndp.Message) bool,
) error {
	for i := 0; ; i++ {
		msg, from, err := sendReceive(ctx, c, m, dst, check)
		switch err {
		case context.Canceled:
			fmt.Println()
			ll.Printf("canceled, sent %d message(s)", i+1)
			return err
		case errRetry:
			fmt.Print(".")
			continue
		case nil:
			fmt.Println()
			printMessage(ll, msg, from)
			return nil
		default:
			return err
		}
	}
}

func receiveLoop(
	ctx context.Context,
	c *ndp.Conn,
	ll *log.Logger,
	check func(m ndp.Message) bool,
	recv func(ll *log.Logger, msg ndp.Message, from netip.Addr),
) error {
	if recv == nil {
		recv = printMessage
	}

	var count int
	for {
		msg, from, err := receive(ctx, c, check)
		switch err {
		case context.Canceled:
			ll.Printf("received %d message(s)", count)
			return nil
		case errRetry:
			continue
		case nil:
			count++
			recv(ll, msg, from)
		default:
			return err
		}
	}
}

var errRetry = errors.New("retry")

func sendReceive(
	ctx context.Context,
	c *ndp.Conn,
	m ndp.Message,
	dst netip.Addr,
	check func(m ndp.Message) bool,
) (ndp.Message, netip.Addr, error) {
	if err := c.WriteTo(m, nil, dst); err != nil {
		return nil, netip.Addr{}, fmt.Errorf("failed to write message: %v", err)
	}

	return receive(ctx, c, check)
}

func receive(
	ctx context.Context,
	c *ndp.Conn,
	check func(m ndp.Message) bool,
) (ndp.Message, netip.Addr, error) {
	if err := c.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		return nil, netip.Addr{}, fmt.Errorf("failed to set deadline: %v", err)
	}

	msg, _, from, err := c.ReadFrom()
	if err == nil {
		if check != nil && !check(msg) {
			// Read a message, but it isn't the one we want.  Keep trying.
			return nil, netip.Addr{}, errRetry
		}

		// Got a message that passed the check, if check was not nil.
		return msg, from, nil
	}

	// Was the context canceled already?
	select {
	case <-ctx.Done():
		return nil, netip.Addr{}, ctx.Err()
	default:
	}

	// Was the error caused by a read timeout, and should the loop continue?
	if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
		return nil, netip.Addr{}, errRetry
	}

	return nil, netip.Addr{}, fmt.Errorf("failed to read message: %v", err)
}
