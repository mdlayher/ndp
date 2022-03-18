// Command ndp is a utility for working with the Neighbor Discovery Protocol.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"os/signal"

	"github.com/mdlayher/ndp"
	"github.com/mdlayher/ndp/internal/ndpcmd"
)

func main() {
	var (
		ifiFlag    = flag.String("i", "", "network interface to use for NDP communication (default: automatic)")
		addrFlag   = flag.String("a", string(ndp.LinkLocal), "address to use for NDP communication (unspecified, linklocal, uniquelocal, global, or a literal IPv6 address)")
		targetFlag = flag.String("t", "", "IPv6 target address for neighbor solicitation NDP messages")
		prefixFlag = flag.String("p", "", "IPv6 address prefix (without CIDR mask) to advertise with router advertisement NDP messages")
	)

	flag.Usage = func() {
		fmt.Println(usage)
		fmt.Println("Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()
	ll := log.New(os.Stderr, "ndp> ", 0)
	if flag.NArg() > 1 {
		ll.Fatalf("too many args on command line: %v", flag.Args()[1:])
	}

	ifi, err := findInterface(*ifiFlag)
	if err != nil {
		ll.Fatalf("failed to get interface: %v", err)
	}

	c, ip, err := ndp.Listen(ifi, ndp.Addr(*addrFlag))
	if err != nil {
		ll.Fatalf("failed to open NDP connection: %v", err)
	}
	defer c.Close()

	var target netip.Addr
	if t := *targetFlag; t != "" {
		target, err = netip.ParseAddr(t)
		if err != nil {
			ll.Fatalf("failed to parse IPv6 target address: %v", err)
		}
	}

	var prefix netip.Addr
	if p := *prefixFlag; p != "" {
		prefix, err = netip.ParseAddr(p)
		if err != nil {
			ll.Fatalf("failed to parse IPv6 prefix address: %v", err)
		}
		if prefix != netip.PrefixFrom(prefix, 64).Masked().Addr() {
			ll.Fatalf("prefix must be a valid /64, got: %q", p)
		}
	}

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigC
		cancel()
	}()

	// Non-Ethernet interfaces (such as PPPoE) may not have a MAC address.
	var mac string
	if ifi.HardwareAddr != nil {
		mac = ifi.HardwareAddr.String()
	} else {
		mac = "none"
	}

	ll.Printf("interface: %s, link-layer address: %s, IPv6 address: %s",
		ifi.Name, mac, ip)

	if err := ndpcmd.Run(ctx, c, ifi, flag.Arg(0), target, prefix); err != nil {
		// Context cancel means a signal was sent, so no need to log an error.
		if err == context.Canceled {
			os.Exit(1)
		}

		ll.Fatal(err)
	}
}

// findInterface attempts to find the specified interface.  If name is empty,
// it attempts to find a usable, up and ready, network interface.
func findInterface(name string) (*net.Interface, error) {
	if name != "" {
		ifi, err := net.InterfaceByName(name)
		if err != nil {
			return nil, fmt.Errorf("could not find interface %q: %v", name, err)
		}

		return ifi, nil
	}

	ifis, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, ifi := range ifis {
		// Is the interface up, multicast, and not a loopback?
		if ifi.Flags&net.FlagUp == 0 ||
			ifi.Flags&net.FlagMulticast == 0 ||
			ifi.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Does the interface have an IPv6 address assigned?
		addrs, err := ifi.Addrs()
		if err != nil {
			return nil, err
		}

		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip, ok := netip.AddrFromSlice(ipNet.IP)
			if !ok {
				panicf("failed to parse IPv6 address: %q", ipNet.IP)
			}

			// Is this address an IPv6 address?
			if ip.Is6() && !ip.Is4In6() {
				return &ifi, nil
			}
		}
	}

	return nil, errors.New("could not find a usable IPv6-enabled interface")
}

const usage = `ndp: utility for working with the Neighbor Discovery Protocol.

Examples:
  Listen for incoming NDP messages on interface eth0 to one of the interface's
  global unicast addresses.

    $ sudo ndp -i eth0 -a global listen
    $ sudo ndp -i eth0 -a 2001:db8::1 listen

  Send router solicitations on interface eth0 from the interface's link-local
  address until a router advertisement is received.

    $ sudo ndp -i eth0 -a linklocal rs

  Send neighbor solicitations on interface eth0 to a neighbor's link-local
  address until a neighbor advertisement is received.

    $ sudo ndp -i eth0 -a linklocal -t fe80::1 ns`

func panicf(format string, a ...any) {
	panic(fmt.Sprintf(format, a...))
}
