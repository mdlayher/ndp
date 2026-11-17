package ndp_test

import (
	"testing"

	"github.com/mdlayher/ndp"
)

func TestCaptivePortalIPHosts(t *testing.T) {
	for _, uri := range []string{
		"http://192.0.2.1/portal", "https://192.0.2.1", "https://192.0.2.1:443/portal",
		"https://[2001:db8::1]/portal", "https://[2001:db8::1]:443/portal",
		"https://[::ffff:192.0.2.1]/portal", "https://[fe80::1%25eth0]/portal",
		"//192.0.2.1/portal", "//[2001:db8::1]/portal",
	} {
		t.Run(uri, func(t *testing.T) {
			if _, err := ndp.NewCaptivePortal(uri); err == nil {
				t.Errorf("NewCaptivePortal(%q) accepted IP host", uri)
			}
			// Explicit construction remains the documented opt-in for IP literals,
			// and received options continue to treat the URI as opaque.
			b, err := ndp.MarshalMessage(&ndp.RouterAdvertisement{Options: []ndp.Option{&ndp.CaptivePortal{URI: uri}, ndp.NewMTU(1500)}})
			if err != nil {
				t.Fatal(err)
			}
			message, err := ndp.ParseMessage(b)
			if err != nil {
				t.Fatal(err)
			}
			options := message.(*ndp.RouterAdvertisement).Options
			if len(options) != 2 || options[0].(*ndp.CaptivePortal).URI != uri || options[1].(*ndp.MTU).MTU != 1500 {
				t.Fatal("explicit URI or following option changed")
			}
		})
	}
	for _, uri := range []string{
		"", ndp.Unrestricted, "https://portal.example.com/portal",
		"https://portal.example.com:8443/portal", "https://192.0.2.1.example.com/portal",
		"https://192.0.2.1@portal.example.com/portal", "https://portal.example.com/portal?ip=192.0.2.1",
	} {
		t.Run(uri, func(t *testing.T) {
			cp, err := ndp.NewCaptivePortal(uri)
			if err != nil {
				t.Fatal(err)
			}
			want := uri
			if want == "" {
				want = ndp.Unrestricted
			}
			if cp.URI != want {
				t.Fatalf("URI=%q,want %q", cp.URI, want)
			}
		})
	}
}
