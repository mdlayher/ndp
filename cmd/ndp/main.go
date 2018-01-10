// Command ndp is a utility for working with the Neighbor Discovery Protocol.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/mdlayher/ndp"
	"github.com/mdlayher/ndp/internal/ndpcmd"
)

func main() {
	var (
		ifiFlag = flag.String("i", "eth0", "network interface to use for NDP communication")
	)

	flag.Usage = func() {
		fmt.Println(usage)
		fmt.Println("Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()
	ll := log.New(os.Stderr, "ndp> ", 0)

	ifi, err := net.InterfaceByName(*ifiFlag)
	if err != nil {
		ll.Fatalf("failed to get interface %q: %v", *ifiFlag, err)
	}

	c, llAddr, err := ndp.Dial(ifi)
	if err != nil {
		ll.Fatalf("failed to dial NDP connection: %v", err)
	}
	defer c.Close()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigC
		cancel()
	}()

	ll.Printf("interface: %s, link-layer address: %s, link-local IPv6 address: %s",
		*ifiFlag, ifi.HardwareAddr, llAddr)

	if err := ndpcmd.Run(ctx, c, ifi, flag.Arg(0)); err != nil {
		// Context cancel means a signal was sent, so no need to log an error.
		if err == context.Canceled {
			os.Exit(1)
		}

		ll.Fatal(err)
	}
}

const usage = `ndp: utility for working with the Neighbor Discovery Protocol.

Examples:
  Send router solicitations on interface eth0 until a router advertisement
  is received.

    $ sudo ndp -i eth0 rs
`
