package main

import (
	"banco_de_dados/pb"
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
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
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 8000+*nodeID))
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
