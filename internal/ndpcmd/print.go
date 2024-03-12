package ndpcmd

import (
	"fmt"
	"io"
	"log"
	"net/netip"
	"strings"

	"github.com/mdlayher/ndp"
)

func printMessage(ll *log.Logger, m ndp.Message, from netip.Addr) {
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

func printRA(ll *log.Logger, ra *ndp.RouterAdvertisement, from netip.Addr) {
	var flags []string
	if ra.ManagedConfiguration {
		flags = append(flags, "managed")
	}
	if ra.OtherConfiguration {
		flags = append(flags, "other")
	}
	if ra.MobileIPv6HomeAgent {
		flags = append(flags, "mobile")
	}
	if ra.NeighborDiscoveryProxy {
		flags = append(flags, "proxy")
	}

	var s strings.Builder
	writef(&s, "router advertisement from: %s:\n", from)

	if ra.CurrentHopLimit > 0 {
		writef(&s, "  - hop limit:        %d\n", ra.CurrentHopLimit)
	}
	if len(flags) > 0 {
		writef(&s, "  - flags:            [%s]\n", strings.Join(flags, ", "))
	}

	writef(&s, "  - preference:       %s\n", ra.RouterSelectionPreference)

	if ra.RouterLifetime > 0 {
		writef(&s, "  - router lifetime:  %s\n", ra.RouterLifetime)
	}
	if ra.ReachableTime != 0 {
		writef(&s, "  - reachable time:   %s\n", ra.ReachableTime)
	}
	if ra.RetransmitTimer != 0 {
		writef(&s, "  - retransmit timer: %s\n", ra.RetransmitTimer)
	}

	_, _ = s.WriteString(optionsString(ra.Options))

	ll.Print(s.String())
}

func printRS(ll *log.Logger, rs *ndp.RouterSolicitation, from netip.Addr) {
	s := fmt.Sprintf(
		rsFormat,
		from.String(),
	)

	ll.Print(s + optionsString(rs.Options))
}

const rsFormat = "router solicitation from %s:\n"

func printNA(ll *log.Logger, na *ndp.NeighborAdvertisement, from netip.Addr) {
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

func printNS(ll *log.Logger, ns *ndp.NeighborSolicitation, from netip.Addr) {
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
	s.WriteString("  - options:\n")

	for _, o := range options {
		writef(&s, "    - %s\n", optStr(o))
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
		return fmt.Sprintf("MTU: %d", o.MTU)
	case *ndp.PrefixInformation:
		var flags []string
		if o.OnLink {
			flags = append(flags, "on-link")
		}
		if o.AutonomousAddressConfiguration {
			flags = append(flags, "autonomous")
		}

		return fmt.Sprintf("prefix information: %s/%d, flags: [%s], valid: %s, preferred: %s",
			o.Prefix.String(),
			o.PrefixLength,
			strings.Join(flags, ", "),
			o.ValidLifetime,
			o.PreferredLifetime,
		)
	case *ndp.RawOption:
		return fmt.Sprintf("type: %03d, value: %v", o.Type, o.Value)
	case *ndp.RouteInformation:
		return fmt.Sprintf("route information: %s/%d, preference: %s, lifetime: %s",
			o.Prefix.String(),
			o.PrefixLength,
			o.Preference.String(),
			o.RouteLifetime,
		)
	case *ndp.RecursiveDNSServer:
		var ss []string
		for _, s := range o.Servers {
			ss = append(ss, s.String())
		}
		servers := strings.Join(ss, ", ")

		return fmt.Sprintf("recursive DNS servers: lifetime: %s, servers: %s", o.Lifetime, servers)
	case *ndp.RAFlagsExtension:
		return fmt.Sprintf("RA flags extension: [%# 02x]", o.Flags)
	case *ndp.DNSSearchList:
		return fmt.Sprintf("DNS search list: lifetime: %s, domain names: %s", o.Lifetime, strings.Join(o.DomainNames, ", "))
	case *ndp.CaptivePortal:
		return fmt.Sprintf("captive portal: %s", o.URI)
	case *ndp.PREF64:
		return fmt.Sprintf("pref64: %s, lifetime: %s", o.Prefix, o.Lifetime)
	case *ndp.Nonce:
		return fmt.Sprintf("nonce: %s", o)
	default:
		panic(fmt.Sprintf("unrecognized option: %v", o))
	}
}

func writef(sw io.StringWriter, format string, a ...any) {
	_, _ = sw.WriteString(fmt.Sprintf(format, a...))
}
