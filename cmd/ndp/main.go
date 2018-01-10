package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/mdlayher/ndp"
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

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigC
		cancel()
	}()

	ifi, err := net.InterfaceByName(*ifiFlag)
	if err != nil {
		ll.Fatalf("failed to get interface %q: %v", *ifiFlag, err)
	}

	c, llAddr, err := ndp.Dial(ifi)
	if err != nil {
		ll.Fatalf("failed to dial NDP connection: %v", err)
	}
	defer c.Close()

	ll.Printf("interface: %s, link-layer address: %s, link-local IPv6 address: %s",
		*ifiFlag, ifi.HardwareAddr, llAddr)

	switch op := flag.Arg(0); op {
	case "rs":
		sendRS(ctx, c, ifi.HardwareAddr)
	default:
		log.Fatalf("unrecognized op: %q", op)
	}
}

func sendRS(ctx context.Context, c *ndp.Conn, addr net.HardwareAddr) {
	ll := log.New(os.Stderr, "ndp rs> ", 0)

	ll.Printf("router solicitation:\n\t- source link-layer address: %s", addr.String())

	m := &ndp.RouterSolicitation{
		Options: []ndp.Option{
			&ndp.LinkLayerAddress{
				Direction: ndp.Source,
				Addr:      addr,
			},
		},
	}

	raC := make(chan *ndp.RouterAdvertisement)

	go func() {
		defer close(raC)
		for {
			msg, _, _, err := c.ReadFrom()
			if err != nil {
				ll.Fatalf("failed to read router advertisement: %v", err)
			}

			ra, ok := msg.(*ndp.RouterAdvertisement)
			if !ok {
				continue
			}

			raC <- ra
			return
		}
	}()

	t := time.NewTicker(1 * time.Second)
	for i := 0; ; i++ {
		if err := c.WriteTo(m, nil, net.IPv6linklocalallrouters); err != nil {
			ll.Fatalf("failed to write router solicitation: %v", err)
		}

		select {
		case <-t.C:
			fmt.Print(".")
		case ra := <-raC:
			printRA(ll, ra)
			return
		case <-ctx.Done():
			fmt.Println()
			ll.Printf("sent %d router solicitation(s)", i+1)
			return
		}
	}
}

func printRA(ll *log.Logger, ra *ndp.RouterAdvertisement) {
	var opts string
	for _, o := range ra.Options {
		opts += fmt.Sprintf("\t\t- %s\n", optStr(o))
	}

	fmt.Println()
	ll.Printf(
		raFormat,
		ra.CurrentHopLimit,
		ra.ManagedConfiguration,
		ra.OtherConfiguration,
		ra.RouterLifetime,
		ra.ReachableTime,
		ra.RetransmitTimer,
		opts,
	)
}

const raFormat = `router advertisement:
	- hop limit: %d
	- managed: %t
	- other: %t
	- router lifetime: %s
	- reachable time: %s
	- retransmit timer: %s
	- options:
%s`

func optStr(o ndp.Option) string {
	switch o := o.(type) {
	case *ndp.LinkLayerAddress:
		dir := "source"
		if o.Direction == ndp.Target {
			dir = "target"
		}

		return fmt.Sprintf("%s link-layer address: %s", dir, o.Addr.String())
	case *ndp.RawOption:
		return fmt.Sprintf("type: %03d, value: %v", o.Type, o.Value)
	default:
		panic(fmt.Sprintf("unrecognized option: %v", o))
	}
}

const usage = `ndp: utility for working with the Neighbor Discovery Protocol.

Examples:
  Send router solicitations on interface eth0 until a router advertisement
  is received.

    $ sudo ndp -i eth0 rs
`
