package main

import (
	"banco_de_dados/pb"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	bolt "go.etcd.io/bbolt"
)

var (
	baseHTTPPort = flag.Uint("baseHTTPPort", 3000, "The base HTTP port")
	baseRPCPort  = flag.Uint("baseRPCPort", 8000, "The base RPC port")
	baseNodeID   = flag.Uint("baseNodeID", 1, "The base node ID")
	nodeCount    = flag.Uint("nodeCount", 10, "The number of nodes in total")
)

type Instance struct {
	NodeID   uint
	HTTPPort uint
	RPCPort  uint

	Logger *log.Logger

	DB *Database

	// RPC
	RPCClients     []pb.DatabaseClient
	RPCClientsLock sync.Mutex

	// CRDT
	PendingCRDTStates     []*pb.DocumentCRDTState
	PendingCRDTStatesLock sync.Mutex
}

func RunInstance(nodeID uint) {
	instance := Instance{
		NodeID:   nodeID,
		HTTPPort: *baseHTTPPort + nodeID,
		RPCPort:  *baseRPCPort + nodeID,
		Logger:   log.New(os.Stdout, fmt.Sprintf("[NODE #%d] ", nodeID), log.LstdFlags),
	}

	instance.openDB()

	go instance.startCRDTTimer()
	go instance.startRPCServer()
	go instance.connectRPCClients()
	go instance.startHTTPServer()
}

func (i *Instance) openDB() {
	db, err := bolt.Open(fmt.Sprintf("./nodes/node_%d", i.NodeID), 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	i.DB = &Database{db: db}
}

func main() {
	flag.Parse()
	wait := make(chan bool)
	for j := *baseNodeID; j < *baseNodeID+*nodeCount; j++ {
		RunInstance(j)
	}
	<-wait
}
