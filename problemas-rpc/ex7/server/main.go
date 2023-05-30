package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/UFGSistemasDistribuidos202301/Grupo4/problemas-rpc/ex7/server/pb"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type server struct {
	pb.UnimplementedAposentadoriaServer
}

func (s *server) PodeAposentar(
	ctx context.Context,
	in *pb.PodeAposentarRequest,
) (*pb.PodeAposentarReply, error) {
	return &pb.PodeAposentarReply{
		PodeAposentar: in.GetIdade() >= 65 || in.GetTempoServico() >= 30 || (in.GetIdade() >= 60 && in.GetTempoServico() >= 25),
	}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterAposentadoriaServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
