package main

import (
	"flag"
)

type config struct {
	RendezvousString string
	ProtocolID       string
	listenHost       string
	listenPort       int
}

func parseFlags() *config {
	c := &config{}

	flag.StringVar(&c.RendezvousString, "rendezvous", "meetme", "Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	flag.StringVar(&c.listenHost, "host", "0.0.0.0", "The bootstrap node host listen address\n")
	flag.StringVar(&c.ProtocolID, "pid", "/DGS/0.0.1", "Sets a protocol id for stream headers")
	flag.IntVar(&c.listenPort, "port", 4422, "node listen port")

	flag.Parse()
	return c
}
