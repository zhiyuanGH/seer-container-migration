package main

import (
	"flag"
	"fmt"
	// "log"

	// "github.com/zhiyuanGH/container-joint-migration/Migration"
	"context"
	pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
	"google.golang.org/grpc"
)

func main() {
	// Define flags for server address and container ID with default values
	serverIp := flag.String("ip", "192.168.116.148", "Server address for container migration")
	containerName := flag.String("container", "loooper2", "ID of the container to migrate")
	serverPort := flag.String("port", "50051", "Server port for container migration")

	// Parse the flags
	flag.Parse()

	// Migrate the container using the provided or default server address and container ID
	addr := *serverIp + ":" + *serverPort
	req := &pb.PullRequest{DestinationIp: *serverIp, DestinationPort: *serverPort, ContainerName: *containerName}
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(200*1024*1024),
	))
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewPullContainerClient(conn)
	res, err := client.PullContainer(context.Background(), req)
	if err != nil {
		fmt.Printf("Container migration failed: %v", err)
	}
	fmt.Printf("New container restored with ID: %s\n", res.ContainerId)

}

