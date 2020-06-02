package ndp

import (
	"bytes"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/ndp/internal/ndptest"
)

func TestConn(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T, c1, c2 *Conn, addr net.IP)
	}{
		{
			name: "echo",
			fn:   testConnEcho,
		},
		{
			name: "filter invalid",
			fn:   testConnFilterInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withConns(t, tt.fn)
		})
	}
}

func testConnEcho(t *testing.T, c1, c2 *Conn, addr net.IP) {
	// Echo this message between two connections.
	rs := &RouterSolicitation{}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		// Read and bounce the message back to the second Conn.
		m, _, _, err := c2.ReadFrom()
		if err != nil {
			panicf("failed to read from c2: %v", err)
		}

		if err := c2.WriteTo(m, nil, addr); err != nil {
			panicf("failed to write from c2: %v", err)
		}
	}()

	if err := c1.WriteTo(rs, nil, addr); err != nil {
		t.Fatalf("failed to write from c1: %v", err)
	}

	m, _, _, err := c1.ReadFrom()
	if err != nil {
		t.Fatalf("failed to read from c1: %v", err)
	}

	wg.Wait()

	if diff := cmp.Diff(rs, m); diff != "" {
		t.Fatalf("unexpected message (-want +got):\n%s", diff)
	}
}

func testConnFilterInvalid(t *testing.T, c1, c2 *Conn, addr net.IP) {
	// Echo this message between two connections.
	rs := &RouterSolicitation{}

	var wg sync.WaitGroup
	wg.Add(1)

	sigC := make(chan struct{})
	go func() {
		defer wg.Done()

		// Wait for the caller to send us a message, then send:
		//  - invalid message (filtered)
		//  - valid message
		// And finally force a timeout to verify the ReadFrom error check.
		m, _, _, err := c2.ReadFrom()
		if err != nil {
			panicf("failed to read from c2: %v", err)
		}

		if err := c2.writeRaw(bytes.Repeat([]byte{0xff}, 255), nil, addr); err != nil {
			panicf("failed to write invalid from c2: %v", err)
		}

		// Write in lockstep and wait for the consumer to acknowledge the write.
		if err := c2.WriteTo(m, nil, addr); err != nil {
			panicf("failed to write valid from c2: %v", err)
		}
		<-sigC

		if err := c1.SetReadDeadline(time.Unix(1, 0)); err != nil {
			panicf("failed to interrupt c1: %v", err)
		}
		<-sigC
	}()

	if err := c1.WriteTo(rs, nil, addr); err != nil {
		t.Fatalf("failed to write from c1: %v", err)
	}

	var m Message
	for i := 0; i < 2; i++ {
		// Acknowledge each write from the other Conn.
		msg, _, _, err := c1.ReadFrom()
		sigC <- struct{}{}

		if err == nil {
			m = msg
			continue
		}

		switch i {
		case 0:
			t.Fatalf("failed to read from c1: %v", err)
		case 1:
			var nerr net.Error
			if !errors.As(err, &nerr) {
				t.Fatalf("error is not net.Error: %v", err)
			}
			if !nerr.Timeout() {
				t.Fatal("error did not indicate a timeout")
			}
		default:
			panic("too many loop iterations")
		}
	}

	wg.Wait()

	if diff := cmp.Diff(rs, m); diff != "" {
		t.Fatalf("unexpected message (-want +got):\n%s", diff)
	}
}

// withConns invokes fn once with a UDPv6 connection and again with an ICMPv6
// connection, enabling testing with both privileged and unprivileged sockets.
func withConns(t *testing.T, fn func(t *testing.T, c1, c2 *Conn, addr net.IP)) {
	var name string
	var newConn func(t *testing.T) (*Conn, *Conn, net.IP, func())

	for i := 0; i < 2; i++ {
		switch i {
		case 0:
			name = "UDP"
			newConn = testUDPConn
		case 1:
			name = "ICMP"
			newConn = testICMPConn
		default:
			t.Fatalf("unhandled withConns iteration: %d", i)
		}

		t.Run(name, func(t *testing.T) {
			c1, c2, addr, done := newConn(t)
			defer done()

			fn(t, c1, c2, addr)
		})
	}
}

func TestSolicitedNodeMulticast(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		snm  net.IP
		ok   bool
	}{
		{
			name: "bad, IPv4",
			ip:   net.IPv4(192, 168, 1, 1),
		},
		{
			name: "ok, link-local",
			ip:   ndptest.MustIPv6("fe80::1234:5678"),
			snm:  ndptest.MustIPv6("ff02::1:ff34:5678"),
			ok:   true,
		},
		{
			name: "ok, global",
			ip:   ndptest.MustIPv6("2001:db8::dead:beef"),
			snm:  ndptest.MustIPv6("ff02::1:ffad:beef"),
			ok:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snm, err := SolicitedNodeMulticast(tt.ip)

			if err != nil && tt.ok {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && !tt.ok {
				t.Fatal("expected an error, but none occurred")
			}
			if err != nil {
				t.Logf("OK error: %v", err)
				return
			}

			if diff := cmp.Diff(tt.snm, snm); diff != "" {
				t.Fatalf("unexpected solicited-node multicast address (-want +got):\n%s", diff)
			}
		})
	}
}
