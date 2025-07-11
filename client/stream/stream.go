package stream

import (
	"context"
	"io"
	"log"

	pb "github.com/Loghadhith/ctx-go/ctxproto"
)

type StreamService struct {
	FilePath  string
	BatchSize int
	Client    pb.StreamFileServiceClient
}

func( s *StreamService) GetListOfAllFiles(ctx context.Context, cancel context.CancelFunc) error {
	req := &pb.DataRequest{}

	list, err := s.Client.GetDataStreaming(ctx,req)
	if err != nil {
		log.Fatalf("List of files rec error: %v",err)
		return err
	}
	log.Println("The list is ",list.Data)
	cancel()
	return nil
}

func( s *StreamService) GetStreamingData(ctx context.Context, cancel context.CancelFunc) error {
	req := &pb.StreamingRequest{Id: "song.flac"}
	stream , err := s.Client.StreamingData(context.Background(),req)
	if err != nil {
		log.Println("Geting the stream data error",err)
	}

	for {
		res, err := stream.Recv()

		if err == io.EOF {
			return nil
		} else if err == nil {
			log.Printf("The response is %d and %s\n",res.Part,res.Data)
		}

		if err != nil {
			log.Println("error in recieving the error",err)
			return err
		}
	}

	cancel()
	return nil
}
