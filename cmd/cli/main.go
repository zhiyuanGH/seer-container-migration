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
	src := flag.String("src", "192.168.116.148:50051", "Server address for source host ")
	dst := flag.String("dst", "192.168.116.149:50051", "Server address for destination host")
	containerName := flag.String("container", "cnn", "ID of the container to migrate")

	// Parse the flags
	flag.Parse()

	// Migrate the container using the provided or default server address and container ID
	
	req := &pb.PullRequest{DestinationAddr: *src, ContainerName: *containerName}

	// should dial the destination server to let it pull container from source server
	conn, err := grpc.Dial(*dst, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
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

