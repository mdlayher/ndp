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

	if err := ndpcmd.Run(ctx, c, ifi, flag.Arg(0), target); err != nil {
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

By default, this tool will automatically bind to IPv6 link-local address of the first interface which is capable of using NDP.

To enable more convenient use without sudo on Linux, apply the CAP_NET_RAW capability:

$ sudo setcap cap_net_raw+ep ./ndp

Examples:
  Listen for incoming NDP messages on the default interface.

    $ ndp

  Send router solicitations on the default interface until a router advertisement is received.

    $ ndp rs

  Send neighbor solicitations on the default interface until a neighbor advertisement is received.

    $ ndp -t fe80::1 ns`

func panicf(format string, a ...any) {
	panic(fmt.Sprintf(format, a...))
}
