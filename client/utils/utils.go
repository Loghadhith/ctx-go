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
	"strconv"
	"strings"
	"sync"

	pb "github.com/Loghadhith/ctx-go/ctxproto"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ClientService struct {
	Addr      string
	FilePath  string
	BatchSize int
	Client    pb.FileUploadServiceClient
}

func FileWithSizeTransfer(path string) ([]byte, []byte) {
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Fatalf("failed to read directory %s: %v", path, err)
	}

	var sizes []string
	var names []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			log.Printf("could not get info for file %s: %v", entry.Name(), err)
			continue
		}
		sizes = append(sizes, fmt.Sprintf("%d", info.Size()))
		names = append(names, entry.Name())
	}

	sizesJoined := strings.Join(sizes, "\n")
	namesJoined := strings.Join(names, "\n")

	// Logging to match original behavior
	log.Println("✅ Files found:", names)
	log.Println("✅ Sizes:", sizes)

	return []byte(sizesJoined), []byte(namesJoined)
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

func DivideAndSendFile(id uint32, stream grpc.ClientStreamingClient[pb.DivideFileUploadReqeust, pb.DivideFileUploadResponse], path string, offset uint32, batchSize uint32) error {
	file, err := os.Open(path)
	if err != nil {
		log.Println("Error opening file:", path)
		return err
	}
	defer file.Close()

	buf := make([]byte, batchSize)
	n, err := file.ReadAt(buf, int64(offset))

	if err != nil && err != io.EOF {
		log.Println("Read error:", err)
		return err
	}
	if n == 0 {
		return errors.New("No bytes read")
	}

	if err := stream.Send(&pb.DivideFileUploadReqeust{
		ChunkId:  uint32(id),
		FileName: path,
		Chunk:    buf,
	}); err != nil {
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

func (s *ClientService) DivideAndSend(ctx context.Context, cancel context.CancelFunc) error {
	sizesBytes, filesBytes := FileWithSizeTransfer(s.FilePath)
	var fileName []string
	fmt.Println("gell", string(filesBytes))
	files := strings.Split(string(filesBytes), "\n")
	sizes := strings.Split(string(sizesBytes), "\n")

	md := metadata.New(map[string]string{})
	md["files"] = files
	md["sizes"] = sizes

	send, _ := metadata.FromOutgoingContext(ctx)
	log.Println("deafult metdata: ", send)
	log.Println("deafult ctx: ", ctx)
	ctx = metadata.NewOutgoingContext(ctx, metadata.Join(send, md))
	log.Println("New ctx: ", ctx)
	log.Println("New metadata: ", md)

	stream, err := s.Client.DivideAndSend(ctx)
	if err != nil {
		log.Printf("Stream error: %v", err)
		return err
	}
	req := &pb.DivideFileUploadReqeust{
		ChunkId:  0,
		FileName: "metadata",
		Chunk:    nil,
	}

	if err := stream.Send(req); err != nil {
		// log.Printf("Send error on chunk %d: %v", chunkID, err)
		log.Println("metadata err: ", err)
		return err
	}
	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Printf("CloseAndRecv error: %v", err)
		return err
	}
	log.Println("Metada succesfully transmitted",res)

	fmt.Println(fileName)
	for _, j := range sizes {
		fmt.Printf("%s", string(j))
	}
	fmt.Println()

	var wg sync.WaitGroup

	for i := range len(files) {
		path := fmt.Sprintf("files/%s", files[i])
		u64, _ := strconv.ParseUint(sizes[i], 10, 32)
		fileSize := uint32(u64)
		wg.Add(1)
		// goroutine for each file
		go func(id int, filePath string, fileSize uint32) {
			defer wg.Done()
			var iwg sync.WaitGroup
			innererrCh := make(chan error, fileSize/uint32(s.BatchSize))

			// var batchnum uint32 = 0

			stream, err := s.Client.DivideAndSend(ctx)
			if err != nil {
				log.Printf("Stream error: %v", err)
				return
			}

			file, err := os.Open(filePath)
			if err != nil {
				log.Printf("File open error: %v", err)
				return
			}
			defer file.Close()

			var chunkID uint32 = 0
			for offset := uint32(0); offset < fileSize; offset += uint32(s.BatchSize) {
				buf := make([]byte, s.BatchSize)
				n, err := file.ReadAt(buf, int64(offset))
				if err != nil && err != io.EOF {
					log.Printf("Read error at offset %d: %v", offset, err)
					return
				}
				if n == 0 {
					break
				}

				req := &pb.DivideFileUploadReqeust{
					ChunkId:  chunkID,
					FileName: filePath,
					Chunk:    buf[:n],
				}

				if err := stream.Send(req); err != nil {
					log.Printf("Send error on chunk %d: %v", chunkID, err)
					return
				}

				log.Printf("✅ Sent chunk %d (%d bytes)", chunkID, n)
				chunkID++
			}

			res, err := stream.CloseAndRecv()
			if err != nil {
				log.Printf("CloseAndRecv error: %v", err)
				return
			}
			log.Printf("🎉 Upload complete: %s (%d bytes confirmed)", res.GetFileName(), res.GetSize())

			iwg.Wait()
			close(innererrCh)

		}(i, path, fileSize)
	}
	wg.Wait()

	return nil
}
