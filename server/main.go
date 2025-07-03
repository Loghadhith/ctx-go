package main

import (
 "context"
 pb "github.com/Loghadhith/ctx-go/ctxproto"
 "google.golang.org/grpc"
 "log"
 "net"
)

type server struct {
 pb.UnimplementedHelloServiceServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
 return &pb.HelloResponse{Message: "Hello, ! "}, nil
}

func main() {
 lis, err := net.Listen("tcp", ":50051")
 if err != nil {
  log.Fatalf("failed to listen on port 50051: %v", err)
 }

 s := grpc.NewServer()
 pb.RegisterHelloServiceServer(s, &server{})
 log.Printf("gRPC server listening at %v", lis.Addr())
 if err := s.Serve(lis); err != nil {
  log.Fatalf("failed to serve: %v", err)
 }
}
