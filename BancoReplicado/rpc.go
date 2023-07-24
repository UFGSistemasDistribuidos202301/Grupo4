package main

import (
	"banco_de_dados/crdt"
	"banco_de_dados/pb"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	bolt "go.etcd.io/bbolt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	rpcClients     = []pb.DatabaseClient{}
	rpcClientsLock = sync.Mutex{}
)

type server struct {
	pb.UnimplementedDatabaseServer
	ID uint
}

func (s *server) MergeCRDTStates(
	ctx context.Context,
	in *pb.MergeCRDTStatesRequest,
) (*pb.MergeCRDTStatesReply, error) {
	log.Printf("Received CRDT sync data")

	err := DB.OpenTx(func(tx *bolt.Tx) error {
		for _, state := range in.Documents {
			receivedCrdtDoc := crdt.MergeableMapFromPB(state.Map)

			table, err := DB.GetTable(tx, state.TableName)
			if err != nil {
				table, err = DB.CreateTable(tx, state.TableName, false)
				if err != nil {
					return err
				}
			}

			crdtTable, ok := table.(*CRDTTable)
			if !ok {
				return errors.New("tried to do a CRDT merge on a non-CRDT table")
			}

			err = crdtTable.Merge(tx, state.DocId, receivedCrdtDoc)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &pb.MergeCRDTStatesReply{}, nil
}

func startRPCServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", rpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterDatabaseServer(s, &server{})
	log.Printf("RPC server listening at %v\n", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func connectRPCClients() {
	rpcClientsLock.Lock()
	defer rpcClientsLock.Unlock()

	for i := *baseNodeID; i < *baseNodeID+*nodeCount; i++ {
		if i == *nodeID {
			continue
		}

		addr := fmt.Sprintf("localhost:%d", *baseRPCPort+i)

		conn, err := grpc.Dial(
			addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		c := pb.NewDatabaseClient(conn)

		log.Printf("RPC connected to node #%d\n", i)

		rpcClients = append(rpcClients, c)
	}
}
