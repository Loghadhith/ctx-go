package file

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	uploadpb "github.com/Loghadhith/ctx-go/ctxproto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FileServiceServer struct {
	uploadpb.UnimplementedFileUploadServiceServer
}

type File struct {
	FilePath   string
	buffer     *bytes.Buffer
	OutputFile *os.File
}

func NewFile() *File {
	return &File{
		buffer: &bytes.Buffer{},
	}
}

func (f *File) SetFile(fileName string) error {
	path := "./upload" // hardcoded location
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		log.Println("Make dir erro")
		return err
	}
	f.FilePath = filepath.Join(path, fileName)
	for i := len(fileName) - 1; i > 0; i-- {
		if fileName[i] == '/' {
			f.FilePath = fileName[i+1:]
		}
	}
	f.FilePath = fmt.Sprintf("%s/%s", path, f.FilePath)
	log.Println(f.FilePath)
	file, err := os.Create(f.FilePath)
	if err != nil {
		log.Println("File create error", err)
		return err
	}
	f.OutputFile = file
	return nil
}

func (f *File) Write(chunk []byte) error {
	if f.OutputFile == nil {
		log.Println("Writ first error")
		return fmt.Errorf("output file is nil")
	}
	_, err := f.OutputFile.Write(chunk)
	// log.Println("the file is written")
	return err
}

func (f *File) Close() error {
	if f.OutputFile != nil {
		return f.OutputFile.Close()
	}
	return nil
}

func (s *FileServiceServer) UploadFile(stream uploadpb.FileUploadService_UploadFileServer) error {
	err := os.Mkdir("upload", 0755)
	if err != nil {
		log.Println("mkdir err", err)
	}
	file := NewFile()
	var fileSize uint32

	defer file.Close()

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("This is the EOF break")
			break
		}
		if err != nil {
			log.Println("This unknow error", err)
			return status.Errorf(codes.Internal, "failed to receive chunk: %v", err)
		}

		if file.FilePath == "" {
			if err := file.SetFile(req.GetFileName()); err != nil {
				return status.Errorf(codes.Internal, "failed to set file: %v", err)
			}
		}

		chunk := req.GetChunk()
		fileSize += uint32(len(chunk))

		if err := file.Write(chunk); err != nil {
			return status.Errorf(codes.Internal, "failed to write chunk: %v", err)
		}
	}

	log.Println("Upload success")
	return stream.SendAndClose(&uploadpb.FileUploadResponse{
		FileName: filepath.Base(file.FilePath),
		Size:     fileSize,
	})
}

func (s *FileServiceServer) DivideAndSend(stream uploadpb.FileUploadService_DivideAndSendServer) error {
	const ChunkSize = 4096

	err := os.MkdirAll("concur", 0755)
	if err != nil {
		log.Println("mkdir error:", err)
		return status.Errorf(codes.Internal, "failed to create directory: %v", err)
	}

	var (
		tmpFile     *os.File
		finalPath   string
		tmpFilePath string
		fileName    string
		totalSize   uint32
		received    = make(map[uint32]bool) // chunk_id -> bool
		mu          sync.Mutex
	)

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("📦 EOF received: Finalizing file...")

			if tmpFile != nil {
				tmpFile.Close()

				if err := os.Rename(tmpFilePath, finalPath); err != nil {
					return status.Errorf(codes.Internal, "failed to rename file: %v", err)
				}
				log.Printf("✅ Upload complete: %s (%d bytes)", finalPath, totalSize)
			}

			return stream.SendAndClose(&uploadpb.DivideFileUploadResponse{
				FileName: fileName,
				Size:     totalSize,
			})
		}
		if err != nil {
			log.Println("receive error:", err)
			return status.Errorf(codes.Internal, "failed to receive chunk: %v", err)
		}

		mu.Lock()

		if tmpFile == nil {
			fileName = filepath.Base(req.GetFileName())
			finalPath = filepath.Join("concur", fileName)
			tmpFilePath = finalPath + ".tmp"

			tmpFile, err = os.OpenFile(tmpFilePath, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				mu.Unlock()
				return status.Errorf(codes.Internal, "failed to create temp file: %v", err)
			}
		}

		chunkID := req.GetChunkId()
		if received[chunkID] {
			log.Printf("⚠️ Skipping already received chunk_id %d", chunkID)
			mu.Unlock()
			continue
		}
		received[chunkID] = true

		chunk := req.GetChunk()
		offset := int64(chunkID) * ChunkSize

		n, err := tmpFile.WriteAt(chunk, offset)
		if err != nil {
			mu.Unlock()
			return status.Errorf(codes.Internal, "failed to write chunk at offset %d: %v", offset, err)
		}

		totalSize += uint32(n)
		mu.Unlock()
	}
}
