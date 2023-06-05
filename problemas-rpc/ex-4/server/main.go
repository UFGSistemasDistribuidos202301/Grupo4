package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/UFGSistemasDistribuidos202301/Grupo4/problemas-rpc/ex-4/server/pb"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type server struct {
	pb.UnimplementedPesoIdealServer
}

func (s *server) CalculaPesoIdeal(
	ctx context.Context,
	in *pb.PesoIdealRequest,
) (*pb.PesoIdealReply, error) {
	var pesoideal float32
	if in.Sexo == "M" {
		pesoideal = 72.7*in.Altura-58
	}else{
		pesoideal = 62.1*in.Altura-44.7
	}
	return &pb.PesoIdealReply{
		PesoIdeal: pesoideal,
	}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterPesoIdealServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
