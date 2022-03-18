package ndp

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"testing"
)

func testICMPConn(t *testing.T) (*Conn, *Conn, netip.Addr) {
	t.Helper()

	ifi := testInterface(t)

	// Create two ICMPv6 connections that will communicate with each other.
	c1, addr := icmpConn(t, ifi)
	c2, _ := icmpConn(t, ifi)

	t.Cleanup(func() {
		_ = c1.Close()
		_ = c2.Close()
	})

	return c1, c2, addr
}

func icmpConn(t *testing.T, ifi *net.Interface) (*Conn, netip.Addr) {
	t.Helper()

	// Wire up a standard ICMPv6 NDP connection.
	c, addr, err := Listen(ifi, LinkLocal)
	if err != nil {
		if !errors.Is(err, os.ErrPermission) {
			t.Fatalf("failed to dial NDP: %v", err)
		}

		t.Skipf("skipping, permission denied, cannot test ICMPv6 NDP: %v", err)
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
			ip, ok := netip.AddrFromSlice(ipNet.IP)
			if !ok {
				t.Fatalf("failed to parse IPv6 address: %v", ipNet.IP)
			}

			// Is this address an IPv6 address?
			if ip.Is6() && !ip.Is4In6() {
				return &ifi
			}
		}
	}

	t.Skip("skipping, could not find a usable IPv6-enabled interface")
	return nil
}
