package main

import (
	"context"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	pb "github.com/Loghadhith/ctx-go/ctxproto"
	uploadpb "github.com/Loghadhith/ctx-go/ctxproto"
	"github.com/Loghadhith/ctx-go/server/file"
)

type helloServer struct {
    pb.UnimplementedHelloServiceServer
}

func (s *helloServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
    log.Println("SayHello called")
    return &pb.HelloResponse{Message: "Hello, ! "}, nil
}

func (s *helloServer) GetHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
    log.Println("GetHello called")
    return &pb.HelloResponse{Message: "This is my first req"}, nil
}

func main() {
		err := os.Mkdir("upload",0755)
		if err != nil {
			log.Println("mkdir err",err)
		}
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    grpcServer := grpc.NewServer()

    pb.RegisterHelloServiceServer(grpcServer, &helloServer{})
    uploadpb.RegisterFileUploadServiceServer(grpcServer, &file.FileServiceServer{})

    log.Printf("gRPC server listening at %v", lis.Addr())
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}

