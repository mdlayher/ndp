package ndpcmd

import (
	"fmt"
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

	s := fmt.Sprintf(
		raFormat,
		from.String(),
		ra.CurrentHopLimit,
		flags,
		ra.RouterSelectionPreference,
		ra.RouterLifetime,
		ra.ReachableTime,
		ra.RetransmitTimer,
	)

	ll.Print(s + optionsString(ra.Options))
}

const raFormat = `router advertisement from: %s:
    - hop limit:        %d
    - flags:            [%s]
    - preference:       %d
    - router lifetime:  %s
    - reachable time:   %s
    - retransmit timer: %s`

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
    - target address: %s`

func printNS(ll *log.Logger, ns *ndp.NeighborSolicitation, from net.IP) {
	s := fmt.Sprintf(
		nsFormat,
		from.String(),
		ns.TargetAddress.String(),
	)

	ll.Print(s + optionsString(ns.Options))
}

const nsFormat = `neighbor solicitation from %s:
    - target address: %s`

func optionsString(options []ndp.Option) string {
	if len(options) == 0 {
		return ""
	}

	var s strings.Builder
	s.WriteString("\n    - options:\n")

	for _, o := range options {
		s.WriteString(fmt.Sprintf("        - %s\n", optStr(o)))
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
