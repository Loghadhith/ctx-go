package main

import (
	"context"
	"log"
	"time"

	"github.com/Loghadhith/ctx-go/client/utils"
	pb "github.com/Loghadhith/ctx-go/ctxproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// conn, err := grpc.DialContext(ctx,
	// 	"localhost:50051",
	// 	grpc.WithTransportCredentials(insecure.NewCredentials()),
	// 	grpc.WithBlock(),
	// )
	// if err != nil {
	// 	log.Fatalf("failed to connect to gRPC server at localhost:50051: %v", err)
	// }
	//
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to gRPC server at localhost:50051: %v", err)
	}
	defer conn.Close()
	log.Println("✅ Connected to gRPC server")

	c := pb.NewHelloServiceClient(conn)

	r, err := c.SayHello(ctx, &pb.HelloRequest{})
	if err != nil {
		log.Fatalf("error calling function SayHello: %v", err)
	}

	log.Printf("Response from gRPC server's SayHello function: %s", r.GetMessage())

	uploadClient := pb.NewFileUploadServiceClient(conn)

	clientService := &utils.ClientService{
		Addr:      "localhost:50051",
		FilePath:  "files",
		BatchSize: 4096,
		Client:    uploadClient,
	}

	uploadCtx, uploadCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer uploadCancel()

	if err := clientService.UploadReader(uploadCtx, uploadCancel); err != nil {
		log.Fatalf("file upload failed: %v", err)
	}
}
