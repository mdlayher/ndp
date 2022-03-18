# CHANGELOG

## v0.9.0

**This is the first release of package `ndp` that only supports Go 1.18+ due to
the use of `net/netip`. Users on older versions of Go must use v0.8.0.**

- [Improvement]: cut over from `net.IP` to `netip.Addr` throughout
- [API Change]: drop `ndp.TestConns`; this API was awkward and didn't test
  actual ICMPv6 functionality. Users are encouraged to either run privileged
  ICMPv6 tests or to swap out `*ndp.Conn` via an interface.
- [Improvement]: drop a lot of awkward test functionality related to
  unprivileged UDP connections to mock out ICMPv6 connections

## v0.8.0

First release of package `ndp` based on the APIs that have been stable for years
with `net.IP`.

**This is the first and last release of package `ndp` which supports Go 1.17 or
older. Future versions will require Go 1.18 and `net/netip`.**
