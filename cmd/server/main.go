package main

import (
    "archive/tar"
    "bytes"
    "compress/gzip"
    "context"
    "fmt"
    "io"
    "log"
    "net"
    "os"
    "path/filepath"
    "time"

    "google.golang.org/grpc"
    pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"

    "github.com/docker/docker/api/types/checkpoint"
    "github.com/docker/docker/client"
)

type server struct {
    pb.UnimplementedContainerMigrationServer
}

func (s *server) CheckpointContainer(ctx context.Context, req *pb.CheckpointRequest) (*pb.CheckpointResponse, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, err
    }

    // Inspect the container to get the full ID
    containerInfo, err := cli.ContainerInspect(ctx, req.ContainerId)
    if err != nil {
        return nil, err
    }
    fullContainerID := containerInfo.ID

    checkpointID := fmt.Sprintf("checkpoint_%d", time.Now().Unix())
    if err := cli.CheckpointCreate(ctx, req.ContainerId, checkpoint.CreateOptions{CheckpointID: checkpointID, Exit: true}); err != nil {
        return nil, err
    }

    checkpointDir := fmt.Sprintf("/var/lib/docker/containers/%s/checkpoints/%s", fullContainerID, checkpointID)
    var buf bytes.Buffer
    gz := gzip.NewWriter(&buf)
    tarWriter := tar.NewWriter(gz)

    err = filepath.Walk(checkpointDir, func(file string, fi os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        hdr, err := tar.FileInfoHeader(fi, file)
        if err != nil {
            return err
        }
        hdr.Name = filepath.ToSlash(file[len(checkpointDir):])
        if err := tarWriter.WriteHeader(hdr); err != nil {
            return err
        }
        if !fi.Mode().IsRegular() {
            return nil
        }
        f, err := os.Open(file)
        if err != nil {
            return err
        }
        defer f.Close()
        if _, err := io.Copy(tarWriter, f); err != nil {
            return err
        }
        return nil
    })
    if err != nil {
        return nil, err
    }

    if err := tarWriter.Close(); err != nil {
        return nil, err
    }
    if err := gz.Close(); err != nil {
        return nil, err
    }

    return &pb.CheckpointResponse{CheckpointId: checkpointID, CheckpointData: buf.Bytes()}, nil
}

func main() {
    lis, err := net.Listen("tcp4", "0.0.0.0:50051") // Use "tcp4" to force IPv4
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterContainerMigrationServer(grpcServer, &server{})
    log.Printf("Server listening at %v", lis.Addr())
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}

//hi
