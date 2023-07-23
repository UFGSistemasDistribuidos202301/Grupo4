package main

import (
	"flag"
	"fmt"
	"log"
)

var (
	nodeID       = flag.Uint("id", 0, "The ID of this node")
	baseHTTPPort = flag.Uint("baseHTTPPort", 3000, "The base HTTP port")
	baseNodeID   = flag.Uint("baseNodeID", 1, "The base node ID")
	nodeCount    = flag.Uint("nodeCount", 1, "The number of nodes in total")
)

func main() {
	log.SetPrefix(fmt.Sprintf("[NODE #%d] ", *nodeID))

	flag.Parse()
	*baseHTTPPort += *nodeID

	openDB()

	go startCRDTTimer()
	go startRPCServer()
	startHTTPServer()
}
