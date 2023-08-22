package main

import (
	"banco_de_dados/crdt"
	"banco_de_dados/pb"
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	bolt "go.etcd.io/bbolt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (i *Instance) MergeCRDTStates(
	ctx context.Context,
	in *pb.MergeCRDTStatesRequest,
) (*pb.MergeCRDTStatesReply, error) {
	if i.isOffline() {
		return nil, errors.New("node is offline")
	}

	i.logger.Printf("Received CRDT sync data")

	err := i.db.OpenTx(func(tx *bolt.Tx) error {
		for _, state := range in.Documents {
			receivedCrdtDoc := crdt.MergeableMapFromPB(state.Map)

			table, err := i.db.GetTable(tx, state.TableName)
			if err != nil {
				table, err = i.db.CreateTable(tx, state.TableName, false)
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

			doc, err := table.Get(tx, state.DocId)
			if err != nil {
				return err
			}

			// Send event
			i.visEventsChannel <- VisEvent{
				NodeID: i.nodeID,
				Kind:   "merge_crdt_states",
				Data:   map[string]any{state.DocId: doc},
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &pb.MergeCRDTStatesReply{}, nil
}

func (i *Instance) QueuePendingCRDTState(
	ctx context.Context,
	in *pb.QueuePendingCRDTStateRequest,
) (*pb.QueuePendingCRDTStateReply, error) {
	pendingStates := i.pendingCRDTStates[uint(in.NodeID)]
	pendingStates = append(pendingStates, in.Document)
	i.pendingCRDTStates[uint(in.NodeID)] = pendingStates
	return &pb.QueuePendingCRDTStateReply{}, nil
}

func (i *Instance) startRPCServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", i.rpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterDatabaseServer(s, i)
	i.logger.Printf("RPC server listening at %v\n", lis.Addr())
	if err := s.Serve(lis); err != nil {
		i.logger.Fatalf("failed to serve: %v", err)
	}
}

func (i *Instance) connectRPCClients() {
	i.rpcClientsLock.Lock()
	defer i.rpcClientsLock.Unlock()

	for j := *baseNodeID; j < *baseNodeID+*nodeCount; j++ {
		if j == i.nodeID {
			continue
		}

		addr := fmt.Sprintf("localhost:%d", *baseRPCPort+j)

		conn, err := grpc.Dial(
			addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		c := pb.NewDatabaseClient(conn)

		i.logger.Printf("RPC connected to node #%d\n", j)

		i.rpcClients[j] = c
	}
}
