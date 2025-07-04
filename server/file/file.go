package file

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

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
		for i := len(fileName)-1 ; i > 0 ; i-- {
			if fileName[i] == '/' {
				f.FilePath = fileName[i+1:]
			}
		}
		f.FilePath = fmt.Sprintf("%s/%s",path,f.FilePath)
		log.Println(f.FilePath)
    file, err := os.Create(f.FilePath)
    if err != nil {
				log.Println("File create error",err)
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
						log.Println("This unknow error",err)
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

