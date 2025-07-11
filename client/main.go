package main

import (
	"context"
	"log"
	"time"

	"github.com/Loghadhith/ctx-go/client/stream"
	"github.com/Loghadhith/ctx-go/client/utils"
	pb "github.com/Loghadhith/ctx-go/ctxproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// conn, err := grpc.NewClient("172.16.101.248:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	// d := pb.NewStreamFileServiceClient(conn)
	// resp, err := d.GetDataStreaming(ctx, &pb.DataRequest{})
	// if err != nil {
	// 	log.Fatalf("Error calling GetDataStreaming: %v", err)
	// }
	// log.Println("GetData")
	// log.Printf("Received file list: %v", resp.Data)

	uploadClient := pb.NewFileUploadServiceClient(conn)

	clientService := &utils.ClientService{
		Addr:      "localhost:50051",
		FilePath:  "files",
		BatchSize: 524288,
		Client:    uploadClient,
	}

	uploadCtx, uploadCancel := context.WithTimeout(context.Background(), time.Second*999)
	defer uploadCancel()

	if err := clientService.DivideAndSend(uploadCtx, uploadCancel); err != nil {
		log.Fatalf("file upload failed: %v", err)
	}

	s := pb.NewStreamFileServiceClient(conn)
	streamService := &stream.StreamService{
		FilePath: "recv",
		BatchSize: 524288,
		Client: s,
	}

	recCtx , recCan := context.WithTimeout(context.Background(), time.Second*10)
	defer recCan()
	streamService.GetListOfAllFiles(recCtx,recCan)

	reCtx , reCan := context.WithTimeout(context.Background(), time.Second*90)
	if err := streamService.GetStreamingData(reCtx,reCan); err != nil {
		log.Fatalf("File receiving error: %v",err)
	}

	// if err := clientService.UploadReader(uploadCtx, uploadCancel); err != nil {
	// 	log.Fatalf("file upload failed: %v", err)
	// }
}
