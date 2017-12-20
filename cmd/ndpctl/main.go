package main

import (
	"log"
	"net"
	"time"

	"github.com/mdlayher/ndp"
)

func main() {
	ifi, err := net.InterfaceByName("eth0")
	if err != nil {
		log.Fatalf("failed to get interface: %v", err)
	}

	c, llAddr, err := ndp.Dial(ifi)
	if err != nil {
		log.Fatalf("failed to dial NDP: %v", err)
	}
	defer c.Close()

	m := &ndp.NeighborAdvertisement{
		TargetAddress: llAddr,
	}

	for i := 0; i < 10; i++ {
		log.Printf("send: %+v", m)

		if err := c.WriteTo(m, nil, nil); err != nil {
			log.Fatalf("failed to write: %v", err)
		}

		time.Sleep(500 * time.Millisecond)
	}
}
