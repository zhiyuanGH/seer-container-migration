package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	exp "github.com/zhiyuanGH/container-joint-migration/exputils"
	pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
	"google.golang.org/grpc"
)

// run on src side, it will run the container, migrate it at random time, and record the p on src, and f on dst
func main() {
	// Define flags for server address and container ID with default values
	src := flag.String("src", "192.168.116.148:50051", "Server address for source host ")
	dst := flag.String("dst", "192.168.116.149:50051", "Server address for destination host")
	containerName := flag.String("container", "cnn", "ID of the container to migrate")
	executor := &exp.RealCommandExecutor{}

	// Parse the flags
	flag.Parse()

	// Migrate the container using the provided or default server address and container ID

	for _, imageName := range containerList {
		for i := 0; i < 5; i++ {
			//Reset the src
			exp.Reset()

			//Extraat the containeralias and write the record file name
			commandArgs, ok := containerCommands[imageName]
			alias, okAlias := containeralias[imageName]
			if !ok || !okAlias {
				log.Printf("No command found for image: %s", imageName)
				continue
			}
			recordPFileName := fmt.Sprintf("/home/base/code/box/data_p/%s_%d.csv", alias, i+1)
			recordFFileName := fmt.Sprintf("/home/base/code/box/data_f/%s_%d.csv", alias, i+1)

			//Run the container on src
			args := append([]string{"docker", "run", "-d", "--name", alias, "-v", "/mnt/nfs_share:/data", imageName}, commandArgs...)
			log.Printf("Executing: sudo %v\n", args)
			_, _, err := executor.Execute(args)
			if err != nil {
				log.Printf("Error during 'docker run': %v", err)
				continue
			}

			//Wait for random time
			log.Printf("Waiting for random time")
			time.Sleep(15 * time.Second)
			log.Printf("Finish Waiting for random time")


			//migrate the container
			req := &pb.PullRequest{DestinationAddr: *src, ContainerName: *containerName, RecordFileName: recordPFileName}
			conn, err := grpc.Dial(*dst, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(200*1024*1024),
			))
			if err != nil {
				fmt.Printf("did not connect: %v\n", err)
			}
			defer conn.Close()

			client := pb.NewPullContainerClient(conn)

			//the PullContainer will trigger the recordP on src
			res, err := client.PullContainer(context.Background(), req)
			if err != nil {
				fmt.Printf("Container migration failed: %v\n", err)
			}
			if res.Success {
				fmt.Printf("New container restored on %s with ID: %s\n", *dst, res.ContainerId)
				//record the F on dst
				recordReq := &pb.RecordRequest{ContainerName: *containerName, RecordFileName: recordFFileName}
				recordClient := pb.NewRecordFClient(conn)
				recordRes, err := recordClient.RecordFReset(context.Background(), recordReq)
				if err != nil {
					fmt.Printf("Record F failed: %v\n", err)
				}
				if recordRes.Success {
					fmt.Printf("Record F success\n")
				}
			}
		}
	}

	// should dial the destination server to let it pull container from source server

}

var containeralias = map[string]string{
	"192.168.116.150:5000/cnn:esgz": "cnn",
}
var containerCommands = map[string][]string{
	"192.168.116.150:5000/cnn:esgz": {"python3", "-u", "main.py", "--batch-size", "64", "--test-batch-size", "1000", "--epochs", "3", "--lr", "0.1", "--gamma", "0.7", "--log-interval", "1", "--save-model"},
}

var containerList = []string{"192.168.116.150:5000/cnn:esgz"}
