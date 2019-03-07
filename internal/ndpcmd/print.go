package ndpcmd

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/mdlayher/ndp"
)

func printMessage(ll *log.Logger, m ndp.Message, from net.IP) {
	switch m := m.(type) {
	case *ndp.NeighborAdvertisement:
		printNA(ll, m, from)
	case *ndp.NeighborSolicitation:
		printNS(ll, m, from)
	case *ndp.RouterAdvertisement:
		printRA(ll, m, from)
	case *ndp.RouterSolicitation:
		printRS(ll, m, from)
	default:
		ll.Printf("%s %#v", from, m)
	}
}

func printRA(ll *log.Logger, ra *ndp.RouterAdvertisement, from net.IP) {
	var flags string
	if ra.ManagedConfiguration {
		flags += "M"
	}
	if ra.OtherConfiguration {
		flags += "O"
	}
	if ra.MobileIPv6HomeAgent {
		flags += "H"
	}
	if ra.NeighborDiscoveryProxy {
		flags += "P"
	}

	var s strings.Builder
	writef(&s, "router advertisement from: %s:\n", from)

	if ra.CurrentHopLimit > 0 {
		writef(&s, "    - hop limit:        %d\n", ra.CurrentHopLimit)
	}
	if flags != "" {
		writef(&s, "    - flags:            [%s]\n", flags)
	}

	writef(&s, "    - preference:       %s\n", ra.RouterSelectionPreference)

	if ra.RouterLifetime > 0 {
		writef(&s, "    - router lifetime:  %s\n", ra.RouterLifetime)
	}
	if ra.ReachableTime != 0 {
		writef(&s, "    - reachable time:   %s\n", ra.ReachableTime)
	}
	if ra.RetransmitTimer != 0 {
		writef(&s, "    - retransmit timer: %s\n", ra.RetransmitTimer)
	}

	_, _ = s.WriteString(optionsString(ra.Options))

	ll.Print(s.String())
}

func printRS(ll *log.Logger, rs *ndp.RouterSolicitation, from net.IP) {
	s := fmt.Sprintf(
		rsFormat,
		from.String(),
	)

	ll.Print(s + optionsString(rs.Options))
}

const rsFormat = `router solicitation from %s:`

func printNA(ll *log.Logger, na *ndp.NeighborAdvertisement, from net.IP) {
	s := fmt.Sprintf(
		naFormat,
		from.String(),
		na.Router,
		na.Solicited,
		na.Override,
		na.TargetAddress.String(),
	)

	ll.Print(s + optionsString(na.Options))
}

const naFormat = `neighbor advertisement from %s:
    - router:         %t
    - solicited:      %t
    - override:       %t
    - target address: %s
`

func printNS(ll *log.Logger, ns *ndp.NeighborSolicitation, from net.IP) {
	s := fmt.Sprintf(
		nsFormat,
		from.String(),
		ns.TargetAddress.String(),
	)

	ll.Print(s + optionsString(ns.Options))
}

const nsFormat = `neighbor solicitation from %s:
    - target address: %s
`

func optionsString(options []ndp.Option) string {
	if len(options) == 0 {
		return ""
	}

	var s strings.Builder
	s.WriteString("    - options:\n")

	for _, o := range options {
		writef(&s, "        - %s\n", optStr(o))
	}

	return s.String()
}

func optStr(o ndp.Option) string {
	switch o := o.(type) {
	case *ndp.LinkLayerAddress:
		dir := "source"
		if o.Direction == ndp.Target {
			dir = "target"
		}

		return fmt.Sprintf("%s link-layer address: %s", dir, o.Addr.String())
	case *ndp.MTU:
		return fmt.Sprintf("MTU: %d", *o)
	case *ndp.PrefixInformation:
		var flags string
		if o.OnLink {
			flags += "O"
		}
		if o.AutonomousAddressConfiguration {
			flags += "A"
		}

		return fmt.Sprintf("prefix information: %s/%d, flags: [%s], valid: %s, preferred: %s",
			o.Prefix.String(),
			o.PrefixLength,
			flags,
			o.ValidLifetime,
			o.PreferredLifetime,
		)
	case *ndp.RawOption:
		return fmt.Sprintf("type: %03d, value: %v", o.Type, o.Value)
	case *ndp.RecursiveDNSServer:
		var ss []string
		for _, s := range o.Servers {
			ss = append(ss, s.String())
		}
		servers := strings.Join(ss, ", ")

		return fmt.Sprintf("recursive DNS servers: lifetime: %s, servers: %s", o.Lifetime, servers)
	case *ndp.DNSSearchList:
		return fmt.Sprintf("DNS search list: lifetime: %s, domain names: %s", o.Lifetime, strings.Join(o.DomainNames, ", "))
	default:
		panic(fmt.Sprintf("unrecognized option: %v", o))
	}
}

func writef(sw io.StringWriter, format string, a ...interface{}) {
	_, _ = sw.WriteString(fmt.Sprintf(format, a...))
}
