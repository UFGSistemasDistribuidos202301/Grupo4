package main

import (
	"banco_de_dados/pb"
	"context"
	"fmt"
	"log"
	"net"
	"sync"

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

func (s *server) RecvCRDTStates(
	ctx context.Context,
	in *pb.RecvCRDTStatesRequest,
) (*pb.RecvCRDTStatesReply, error) {
	for _, table := range in.Tables {
		for _, doc := range table.GetDocs() {
			_ = doc
			_ = table
		}
	}
	return &pb.RecvCRDTStatesReply{}, nil
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
