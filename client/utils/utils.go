package utils

import (
	"bufio"
	"context"
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

func FileWithSizeTransfer(path string) {
	str := "awk '{print $2,$8}'"
	path = fmt.Sprintf("l %s | %s", path,str)
	// cmd := exec.Command("bash", "-c", "ls | grep flac")
	cmd := exec.Command("bash", "-c", path)
	fmt.Println(cmd)
	data, _ := cmd.Output()
	log.Println(string(data))
	for _, v := range data {
		fmt.Print(string(v))
	}
	fmt.Println()
}

func FileToTransfer(path string) ([]string, error) {
	path = fmt.Sprintf("ls %s", path)
	// cmd := exec.Command("bash", "-c", "ls | grep flac")
	cmd := exec.Command("bash", "-c", path)
	fmt.Println(cmd)
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

func SendFileReq(id int, stream grpc.ClientStreamingClient[pb.FileUploadRequest, pb.FileUploadResponse], path string) error {
	file, err := os.Open(path)
	if err != nil {
		log.Println("Error opening file:", path)
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buf := make([]byte, 1024)

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
