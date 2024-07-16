package main

import (
    "context"
    "fmt"
    "io/ioutil"
    "log"
    "net"
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

    checkpointID := fmt.Sprintf("checkpoint_%d", time.Now().Unix())
    if err := cli.CheckpointCreate(ctx, req.ContainerId, checkpoint.CreateOptions{CheckpointID: checkpointID, Exit: true}); err != nil {
        return nil, err
    }

    checkpointDir := fmt.Sprintf("/var/lib/docker/containers/%s/checkpoints/%s", req.ContainerId, checkpointID)
    checkpointData, err := ioutil.ReadFile(checkpointDir + "/checkpoint.tar")
    if err != nil {
        return nil, err
    }

    return &pb.CheckpointResponse{CheckpointId: checkpointID, CheckpointData: checkpointData}, nil
}

func main() {
    lis, err := net.Listen("tcp", ":50051")
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
