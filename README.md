ndp [![Build Status](https://travis-ci.org/mdlayher/ndp.svg?branch=master)](https://travis-ci.org/mdlayher/ndp) [![GoDoc](https://godoc.org/github.com/mdlayher/ndp?status.svg)](https://godoc.org/github.com/mdlayher/ndp) [![Go Report Card](https://goreportcard.com/badge/github.com/mdlayher/ndp)](https://goreportcard.com/report/github.com/mdlayher/ndp)
===

Package `ndp` implements the Neighbor Discovery Protocol, as described in
[RFC 4861](https://tools.ietf.org/html/rfc4861).  MIT Licensed.

The command `ndp` is a utility for working with the Neighbor Discovery Protocol.

## Examples

Listen for incoming NDP messages on interface eth0 to one of the interface's
global unicast addresses.
    
```
$ sudo ndp -i eth0 -a global listen
$ sudo ndp -i eth0 -a 2001:db8::1 listen
````

Send router solicitations on interface eth0 from the interface's link-local
address until a router advertisement is received.
    
```
$ sudo ndp -i eth0 -a linklocal rs
```
