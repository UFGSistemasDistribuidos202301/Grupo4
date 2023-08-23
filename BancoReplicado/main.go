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
	pb.UnimplementedDatabaseServer

	nodeID   uint
	httpPort uint
	rpcPort  uint

	logger *log.Logger

	db *Database

	// RPC
	rpcClients     map[uint]pb.DatabaseClient
	rpcClientsLock sync.Mutex

	// CRDT
	pendingCRDTStates     map[uint][]*pb.DocumentCRDTState
	pendingCRDTStatesLock sync.Mutex
	timerDisabledMutex sync.RWMutex
	timerDisabled      bool

	// WebSocket events
	wsListeners      map[*websocket.Conn]chan<- VisEvent
	wsListenersLock  sync.Mutex
	visEventsChannel chan VisEvent

	// Offline simulation
	offlineMutex sync.RWMutex
	offline      bool

}

func RunInstance(nodeID uint) {
	instance := Instance{
		nodeID:   nodeID,
		httpPort: *baseHTTPPort + nodeID,
		rpcPort:  *baseRPCPort + nodeID,
		logger:   log.New(os.Stdout, fmt.Sprintf("[NODE #%d] ", nodeID), log.LstdFlags),

		rpcClients:        make(map[uint]pb.DatabaseClient),
		pendingCRDTStates: make(map[uint][]*pb.DocumentCRDTState),

		wsListeners:      make(map[*websocket.Conn]chan<- VisEvent),
		visEventsChannel: make(chan VisEvent),
	}

	instance.openDB()

	go instance.startCRDTTimer()
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

func (i *Instance) isOffline() bool {
	i.offlineMutex.RLock()
	offline := i.offline
	i.offlineMutex.RUnlock()
	return offline
}

func (i *Instance) isCRDTTimerEnabled() bool {
	i.timerDisabledMutex.RLock()
	disabled := i.timerDisabled
	i.timerDisabledMutex.RUnlock()
	return !disabled
}

func main() {
	flag.Parse()
	wait := make(chan bool)
	for j := *baseNodeID; j < *baseNodeID+*nodeCount; j++ {
		RunInstance(j)
	}
	<-wait
}
