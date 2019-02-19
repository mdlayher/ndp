package ndp

import (
	"fmt"
	"net"
	"os"
	"testing"
)

func testICMPConn(t *testing.T) (*Conn, *Conn, net.IP, func()) {
	ifi := testInterface(t)

	// Create two ICMPv6 connections that will communicate with each other.
	c1, addr := icmpConn(t, ifi)
	c2, _ := icmpConn(t, ifi)

	return c1, c2, addr, func() {
		_ = c1.Close()
		_ = c2.Close()
	}
}

func testUDPConn(t *testing.T) (*Conn, *Conn, net.IP, func()) {
	ifi := testInterface(t)

	c1, c2, ip, err := TestConns(ifi)
	if err != nil {
		// TODO(mdlayher): remove when travis can do IPv6.
		t.Skipf("failed to create test connections, skipping test: %v", err)
	}

	return c1, c2, ip, func() {
		_ = c1.Close()
		_ = c2.Close()
	}
}

func icmpConn(t *testing.T, ifi *net.Interface) (*Conn, net.IP) {
	// Wire up a standard ICMPv6 NDP connection.
	c, addr, err := Dial(ifi, LinkLocal)
	if err != nil {
		oerr, ok := err.(*net.OpError)
		if !ok && (ok && !os.IsPermission(err)) {
			t.Fatalf("failed to dial NDP: %v", err)
		}

		t.Skipf("permission denied, cannot test ICMPv6 NDP: %v", oerr)
	}
	c.icmpTest = true

	return c, addr
}

func testInterface(t *testing.T) *net.Interface {
	t.Helper()

	ifis, err := net.Interfaces()
	if err != nil {
		t.Fatalf("failed to get interfaces: %v", err)
	}

	for _, ifi := range ifis {
		// Is the interface up and not a loopback?
		if ifi.Flags&net.FlagUp != 1 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Does the interface have an IPv6 address assigned?
		addrs, err := ifi.Addrs()
		if err != nil {
			t.Fatalf("failed to get interface %q addresses: %v", ifi.Name, err)
		}

		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}

			// Is this address an IPv6 address?
			if ipNet.IP.To16() != nil && ipNet.IP.To4() == nil {
				return &ifi
			}
		}
	}

	t.Skip("could not find a usable IPv6-enabled interface")
	return nil
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
