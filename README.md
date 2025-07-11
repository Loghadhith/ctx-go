# ctx-go

A high-performance gRPC-based file transfer system in Go, supporting concurrent, chunked uploads of large files between client and server.

## Features

- **Concurrent File Uploads:** Transfer multiple files in parallel.
- **Chunked Streaming:** Large files are split into configurable-size chunks for efficient streaming.
- **gRPC Protocol:** Uses gRPC streaming for robust, scalable, and type-safe communication.
- **Customizable Chunk Size:** Tune chunk size for optimal performance on your network and hardware.
- **Server-Side Preallocation:** Server preallocates files for efficient random-access writes.

## Project Structure

```
ctx-go/
  client/         # Client code for uploading files
  server/         # Server code for receiving files
  ctxproto/       # Protobuf definitions
```

## Getting Started

### Prerequisites

- Go 1.20+
- Protobuf compiler (`protoc`)
- (Optional) [Nix](https://nixos.org/) for reproducible dev environments

### Installation

1. **Clone the repository:**
   ```sh
   git clone https://github.com/yourusername/ctx-go.git
   cd ctx-go
   ```

2. **Generate gRPC code:**
   ```sh
   protoc --go_out=. --go-grpc_out=. ctxproto/ctxproto.proto
   ```

3. **Build the server and client:**
   ```sh
   go build -o server/server ./server
   go build -o client/client ./client
   ```

### Usage

#### 1. Start the Server

```sh
cd server
go run main.go
```

The server listens on `localhost:50051` by default.

#### 2. Run the Client

```sh
cd client
go run main.go
```

- Place files to upload in the `client/files/` directory.
- The client will upload all files in that directory to the server.

### Configuration

- **Chunk Size:**  
  Set the `BatchSize` field in `client/main.go` or via environment variable for optimal performance (e.g., `262144` for 256 KB).
- **Directories:**  
  - Client reads from `client/files/`
  - Server writes to `server/concur/` (for concurrent) or `server/upload/` (for sequential)

### Protocol

- See `ctxproto/ctxproto.proto` for message and service definitions.

## Performance Tuning

- For local networks, a chunk size of **256 KB to 1 MB** is recommended.
- Use SSDs for best disk I/O performance.
- Monitor CPU and memory usage with Go’s built-in `pprof` (enabled on client).

---
