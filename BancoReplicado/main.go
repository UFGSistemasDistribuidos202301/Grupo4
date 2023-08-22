package main

import (
	"banco_de_dados/pb"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	bolt "go.etcd.io/bbolt"
)

var (
	baseHTTPPort = flag.Uint("baseHTTPPort", 3000, "The base HTTP port")
	baseRPCPort  = flag.Uint("baseRPCPort", 8000, "The base RPC port")
	baseNodeID   = flag.Uint("baseNodeID", 1, "The base node ID")
	nodeCount    = flag.Uint("nodeCount", 10, "The number of nodes in total")
)

type Instance struct {
	nodeID   uint
	httpPort uint
	rpcPort  uint

	logger *log.Logger

	db *Database

	// RPC
	rpcClients     []pb.DatabaseClient
	rpcClientsLock sync.Mutex

	// CRDT
	pendingCRDTStates     []*pb.DocumentCRDTState
	pendingCRDTStatesLock sync.Mutex

	// WebSocket events
	wsListeners      map[*websocket.Conn]chan<- VisEvent
	wsListenersLock  sync.Mutex
	visEventsChannel chan VisEvent
}

func RunInstance(nodeID uint) {
	instance := Instance{
		nodeID:   nodeID,
		httpPort: *baseHTTPPort + nodeID,
		rpcPort:  *baseRPCPort + nodeID,
		logger:   log.New(os.Stdout, fmt.Sprintf("[NODE #%d] ", nodeID), log.LstdFlags),

		wsListeners:      make(map[*websocket.Conn]chan<- VisEvent),
		visEventsChannel: make(chan VisEvent),
	}

	instance.openDB()

	// go instance.startCRDTTimer()
	go instance.startRPCServer()
	go instance.connectRPCClients()
	go instance.startHTTPServer()
}

func (i *Instance) openDB() {
	db, err := bolt.Open(fmt.Sprintf("./nodes/node_%d", i.nodeID), 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	i.db = &Database{db: db, instance: i}
}

func main() {
	flag.Parse()
	wait := make(chan bool)
	for j := *baseNodeID; j < *baseNodeID+*nodeCount; j++ {
		RunInstance(j)
	}
	<-wait
}
