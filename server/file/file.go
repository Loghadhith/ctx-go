package file

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	uploadpb "github.com/Loghadhith/ctx-go/ctxproto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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

func WriteTmpFil(files []string, sizes []string) {
	// Createing each file with tmp data
	for i := 0; i < len(files); i++ {
		size, err := strconv.ParseInt(sizes[i], 10, 64)
		if err != nil {
			log.Printf("Invalid size for file %s: %v", files[i], err)
			continue
		}

		// Create or truncate the file
		path := fmt.Sprintf("concur/%s", files[i])
		f, err := os.Create(path)
		// d := []byte("Hello damn")
		// f.Write(d)
		if err != nil {
			log.Printf("Error creating file %s: %v", files[i], err)
			continue
		}

		// Preallocate the file size with zeros
		// Option 1: Use Truncate to set file size (content will be zero-filled on most systems)
		err = f.Truncate(size)
		if err != nil {
			log.Printf("Error truncating file %s to size %d: %v", files[i], size, err)
			f.Close()
			continue
		}

		// Close the file after allocation
		err = f.Close()
		if err != nil {
			log.Printf("Error closing file %s: %v", files[i], err)
		}

		// log.Printf("Created temp file %s with size %d bytes", files[i], size)
		// log.Println("Metadata", files[i])
	}
	log.Println("Created the tmp files with dummy data")
}

func (s *FileServiceServer) DivideAndSend(stream uploadpb.FileUploadService_DivideAndSendServer) error {
	const ChunkSize = 524288

	var (
		received  = make(map[uint32]bool) // chunk_id -> bool
		totalSize uint32
		File      *os.File
		fileName  string
	)

	for {
		req, err := stream.Recv()
		fileName = req.GetFileName()

		if err == io.EOF {
			log.Println("📦 EOF received: Finalizing file...")

			return stream.SendAndClose(&uploadpb.DivideFileUploadResponse{
				FileName: fileName,
				Size:     totalSize,
			})
		}
		// log.Println("req 1st", req)
		// log.Println("req rahul", req.GetFileName())
		// log.Println("stream ", stream.Context())
		if req.GetFileName() == "metadata" {
			//do something with metadata
			md, ok := metadata.FromIncomingContext(stream.Context())
			if !ok {
				log.Println("Metadata parse error: ", ok)
			}

			// log.Println("This is my metadata", md)
			// log.Println(md["files"], md["sizes"])
			WriteTmpFil(md["files"], md["sizes"])
			// log.Println("skip", req.GetChunkId())
			continue
		}
		if err != nil {
			log.Println("receive error:", err)
			return status.Errorf(codes.Internal, "failed to receive chunk: %v", err)
		}
		ChunkID := req.ChunkId
		if received[ChunkID] {
			log.Printf("⚠️ Skipping already received chunk_id %d", ChunkID)
			continue
		}
		received[ChunkID] = true

		Chunk := req.GetChunk()
		offset := int64(ChunkID) * ChunkSize
		fileName := req.FileName
		// log.Println("Inga mudila")
		File, err = os.OpenFile(fmt.Sprintf("concur/%s", fileName), os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "failed to open the file")
		}

		// log.Println("Inga mudila")
		n, err := File.WriteAt(Chunk, offset)
		// log.Println("Inga solee mudinchu")
		if err != nil {
			return status.Errorf(codes.Internal, "failed to write chunk at offset %d: %v", offset, err)
		}

		totalSize += uint32(n)
	}
}
