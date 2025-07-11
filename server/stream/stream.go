package stream

import (
	"context"
	"fmt"
	"log"
	"os"

	pb "github.com/Loghadhith/ctx-go/ctxproto"
)

type StreamingServiceServer struct {
	pb.UnimplementedStreamFileServiceServer
}

func StreamTheFile(file *os.File) {
	const chunkSize = 524288

	Info, _ := file.Stat()
	for offset := uint32(0); offset < uint32(Info.Size()); offset += uint32(chunkSize) {
		log.Println(offset)
		// Do something with the file
	}
}

func FilesToStream() ([]string, error) {
	dir, err := os.ReadDir("concur")
	if err != nil {
		log.Println("Directory error", err)
		return nil, err
	}
	var names []string
	for _, entry := range dir {
		if entry.IsDir() {
			continue
		}
		_, err := entry.Info()
		if err != nil {
			log.Printf("could not get info for file %s: %v", entry.Name(), err)
			continue
		}
		names = append(names, entry.Name())
	}
	return names, nil
}

func (s *StreamingServiceServer) GetDataStreaming(ctx context.Context, req *pb.DataRequest) (*pb.DataResponse, error) {
	files, err := FilesToStream()
	if err != nil {
		return nil, err
	}
	log.Println("GetDataStreaming is called")

	return &pb.DataResponse{Data: files}, nil
}

func (s *StreamingServiceServer) StreamingData(req *pb.StreamingRequest, stream pb.StreamFileService_StreamingDataServer) error {
	const chunk_size = 524288
	file, err := os.OpenFile(fmt.Sprintf("concur/%s",req.GetId()), os.O_RDONLY, 0)
	Info, _ := file.Stat()
	if err != nil {
		log.Println("Opening the file error",err)
		return err
	}

	log.Println("Streaming file: ",Info.Name())
	var batch_num uint32
	for offset := uint32(0) ; int64(offset) < Info.Size() ; offset+=uint32(chunk_size) {
		buf := make([]byte , chunk_size)
		n, _ := file.ReadAt(buf,int64(offset))
		resp := &pb.StreamingResponse{
			Part: batch_num,
			Data: Info.Name(),
			Chunk: buf[:n],
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
		batch_num++
	}
	return nil
}
