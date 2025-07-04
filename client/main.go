package main

import (
	"context"
	"log"
	"time"

	pb "github.com/Loghadhith/ctx-go/ctxproto"
	"github.com/Loghadhith/ctx-go/client/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)


func main() {
 conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
 if err != nil {
  log.Fatalf("failed to connect to gRPC server at localhost:50051: %v", err)
 }
 defer conn.Close()
 c := pb.NewHelloServiceClient(conn)

 ctx, cancel := context.WithTimeout(context.Background(), time.Second)
 defer cancel()

 r, err := c.SayHello(ctx, &pb.HelloRequest{})
 if err != nil {
  log.Fatalf("error calling function SayHello: %v", err)
 }

 log.Printf("Response from gRPC server's SayHello function: %s", r.GetMessage())

 // fill , _ := utils.FileToTransfer()
 // log.Println(fill)
	
 uploadClient := pb.NewFileUploadServiceClient(conn)

	clientService := &utils.ClientService{
		Addr:      "localhost:50051",     // optional, not used here
		FilePath:  "files",            // path to file you want to upload
		BatchSize: 1024,                  // size in bytes of each chunk
		Client:    uploadClient,
	}

	uploadCtx, uploadCancel := context.WithTimeout(context.Background(), time.Second*200)
	defer uploadCancel()

	if err := clientService.UploadReader(uploadCtx, uploadCancel); err != nil {
		log.Fatalf("file upload failed: %v", err)
	}
}
