package utils

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	pb "github.com/Loghadhith/ctx-go/ctxproto"
	grpc "google.golang.org/grpc"
)

type ClientService struct {
	Addr      string
	FilePath  string
	BatchSize int
	Client    pb.FileUploadServiceClient
}

func FileWithSizeTransfer(path string) ([]byte,[]byte) {
	siz := "awk '{print $5}'"
	sizPath := fmt.Sprintf("ls -l %s | %s", path, siz)

	file := "awk '{print $9}'"
	filePath := fmt.Sprintf("ls -l %s | %s", path, file)

	cmdSiz := exec.Command("bash", "-c", sizPath)
	cmdFile := exec.Command("bash", "-c", filePath)
	dataS, _ := cmdSiz.Output()
	log.Println("string chk", string(dataS), len(dataS))
	log.Println("ok\n")

	dataF, _ := cmdFile.Output()
	log.Println("string chk", string(dataF), len(dataF))
	log.Println("ok\n")

	fmt.Print("The data is : ")
	for _, v := range dataS {
		fmt.Print(string(v))
	}
	fmt.Println()

	fmt.Print("The data is : ")
	for _, v := range dataF {
		fmt.Print(string(v))
	}
	fmt.Println()
	return dataS,dataF
}

func FileToTransfer(path string) ([]string, error) {
	path = fmt.Sprintf("ls %s", path)
	cmd := exec.Command("bash", "-c", path)
	data, err := cmd.Output()
	log.Println(string(data))
	for _, v := range data {
		fmt.Print(string(v))
	}
	fmt.Println()

	if err != nil {
		// Check if it's just "no match found"
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			log.Println("No .flac files found.")
			return nil, err
		}
		log.Println("Command failed:", err)
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	return lines, nil
}

func DivideAndSendFile(id uint32, stream grpc.ClientStreamingClient[pb.DivideFileUploadReqeust, pb.DivideFileUploadResponse], path string,offset uint32,batchSize uint32) error {
	file, err := os.Open(path)
	if err != nil {
		log.Println("Error opening file:", path)
		return err
	}
	defer file.Close()

	buf := make([]byte, batchSize)
	n, err := file.ReadAt(buf,int64(offset))

	if err != nil && err != io.EOF {
		log.Println("Read error:", err)
		return err
	}
	if n == 0 {
		return errors.New("No bytes read")
	}

	if err := stream.Send(&pb.DivideFileUploadReqeust{
		ChunkId: uint32(id),
		FileName: path,
		Chunk: buf,
	});	err != nil {
		log.Println("Send error:", err)
		return err
	}

	if err == io.EOF {
		return err
	}
	log.Println("✅ File upload succeeded:", path)
	return nil
}

func SendFileReq(id int, stream grpc.ClientStreamingClient[pb.FileUploadRequest, pb.FileUploadResponse], path string) error {
	file, err := os.Open(path)
	if err != nil {
		log.Println("Error opening file:", path)
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	// prev bottleneck faced when transferring to sp and mk
	buf := make([]byte, 4096)

	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			log.Println("Read error:", err)
			return err
		}
		if n == 0 {
			break
		}

		if err := stream.Send(&pb.FileUploadRequest{
			FileName: path,
			Chunk:    buf[:n],
		}); err != nil {
			log.Println("Send error:", err)
			return err
		}

		if err == io.EOF {
			break
		}
	}

	log.Println("✅ File upload succeeded:", path)
	return nil
}


func (s *ClientService) UploadReader(ctx context.Context, cancel context.CancelFunc) error {
	FileWithSizeTransfer(s.FilePath)
	str, etrErr := FileToTransfer(s.FilePath)
	if etrErr != nil {
		log.Println("FileToTransfer error:", etrErr)
		return etrErr
	}
	log.Println("Files to upload:", str)

	var wg sync.WaitGroup
	errCh := make(chan error, len(str))

	for i, path := range str {
		path := fmt.Sprintf("files/%s", path)
		wg.Add(1)
		go func(id int, filePath string) {
			defer wg.Done()

			stream, err := s.Client.UploadFile(ctx)
			if err != nil {
				errCh <- fmt.Errorf("stream start error: %w", err)
				return
			}

			if err := SendFileReq(id, stream, filePath); err != nil {
				errCh <- fmt.Errorf("send file error: %w", err)
				return
			}

			res, err := stream.CloseAndRecv()
			if err != nil {
				errCh <- fmt.Errorf("close and recv error: %w", err)
				return
			}

			log.Printf("✅ Sent - %v bytes - %s\n", res.GetSize(), res.GetFileName())
		}(i, path)
	}

	wg.Wait()
	close(errCh)

	cancel()

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

func(s *ClientService) DivideAndSend(ctx context.Context, cancel context.CancelFunc) error {
	sizes , files := FileWithSizeTransfer(s.FilePath)


	var wg sync.WaitGroup

	for i := range len(files) {
		path := fmt.Sprintf("files/%s",files[i])
		fileSize := uint32(sizes[i])
		wg.Add(1)
		// goroutine for each file
		go func (id int , filePath string, fileSize uint32){
			defer wg.Done()
			var iwg sync.WaitGroup
			innererrCh := make(chan error, fileSize/uint32(s.BatchSize))

			var batchnum uint32 = 0
			for j := uint32(0) ; uint32(j) < fileSize ; j += uint32(s.BatchSize) {
				iwg.Add(1)
				go func(name string, ttsize uint32,offset uint32,id uint32) {
					defer iwg.Done()
					stream, err := s.Client.DivideAndSend(ctx)
					if err != nil {
						return
					}

					if err := DivideAndSendFile(batchnum,stream,name,j,uint32(s.BatchSize)); err != nil {
						innererrCh <- fmt.Errorf("send file error: %w", err)
						return
					}

					res, err := stream.CloseAndRecv()
					if err != nil {
						innererrCh <- fmt.Errorf("close and recv error: %w", err)
						return
					}
					batchnum++
					log.Printf("✅ Sent - %v bytes - %s\n", res.GetSize(), res.GetFileName())
				}(filePath,fileSize,j,uint32(batchnum))
			}
			iwg.Wait()
			close(innererrCh)

		}(i,path,fileSize)
	}
	wg.Wait()

	return nil
}
