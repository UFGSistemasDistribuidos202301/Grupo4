package main

import (
	"flag"
	"fmt"
	"log"
)

var (
	nodeID       = flag.Uint("id", 0, "The ID of this node")
	baseHTTPPort = flag.Uint("baseHTTPPort", 3000, "The base HTTP port")
	baseRPCPort  = flag.Uint("baseRPCPort", 8000, "The base RPC port")
	baseNodeID   = flag.Uint("baseNodeID", 1, "The base node ID")
	nodeCount    = flag.Uint("nodeCount", 1, "The number of nodes in total")

	httpPort uint
	rpcPort  uint
)

func main() {
	flag.Parse()

	httpPort = *baseHTTPPort + *nodeID
	rpcPort = *baseRPCPort + *nodeID

	log.SetPrefix(fmt.Sprintf("[NODE #%d] ", *nodeID))

	openDB()

	go startCRDTTimer()
	go startRPCServer()
	go connectRPCClients()
	startHTTPServer()
}
