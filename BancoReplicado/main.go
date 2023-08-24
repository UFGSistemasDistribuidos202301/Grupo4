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

	colors = []string{
		"\033[31m",
		"\033[32m",
		"\033[33m",
		"\033[34m",
		"\033[35m",
		"\033[36m",
		"\033[37m",
		"\033[91m",
		"\033[92m",
		"\033[93m",
		"\033[94m",
		"\033[95m",
		"\033[96m",
		"\033[97m",
	}
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
	color := colors[(int(nodeID)-1)%len(colors)]
	instance := Instance{
		nodeID:   nodeID,
		httpPort: *baseHTTPPort + nodeID,
		rpcPort:  *baseRPCPort + nodeID,
		logger:   log.New(os.Stdout, fmt.Sprintf("%s[NODE #%d] ", color, nodeID), log.LstdFlags),

		rpcClients: make(map[uint]pb.DatabaseClient),

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
