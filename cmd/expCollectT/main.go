package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"encoding/csv"
	"os"
	"time"

	exp "github.com/zhiyuanGH/container-joint-migration/exputils"
	pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
	"google.golang.org/grpc"
)

func main() {
	// Define flags for server address and container ID with default values
	src := flag.String("src", "192.168.116.148:50051", "Server address for source host ")
	dst := flag.String("dst", "192.168.116.149:50051", "Server address for destination host")
	executor := &exp.RealCommandExecutor{}
	fmt.Println("Testing")
	csvFilePath := "/home/base/code/box/data_t/data.csv"


	// Parse the flags
	flag.Parse()

	// Define flags for each image (can add more flags as needed)
	imageFlags := map[string][]string{
		"192.168.116.150:5000/cnn:esgz":      {"-v", "/mnt/nfs_share:/data"}, // Example of port flag for cnn image
		"192.168.116.150:5000/node:esgz":     {"-p", "8080:80"},              // Example of port flag for node image
		"192.168.116.150:5000/postgres:esgz": {},
	}

	// Migrate the container using the provided or default server address and container ID
	for _, imageName := range containerList {
		for i := 0; i < 3; i++ {
			// Reset the src
			exp.Reset()

			// Extract the container alias and write the record file name
			commandArgs, ok := containerCommands[imageName]
			alias, okAlias := containeralias[imageName]
			if !ok || !okAlias {
				log.Printf("No command found for image: %s", imageName)
				continue
			}

			// Get the specific flags for the current image
			imageSpecificFlags, ok := imageFlags[imageName]
			if !ok {
				log.Printf("No specific flags found for image: %s", imageName)
				continue
			}

			// Run the container on src
			args := append([]string{"docker", "run", "-d", "--name", alias}, imageSpecificFlags...)
			args = append(args, imageName)
			args = append(args, commandArgs...)
			log.Printf("Executing: sudo %v\n", args)
			_, _, err := executor.Execute(args)
			if err != nil {
				log.Printf("Error during 'docker run': %v", err)
				continue
			}

			// Wait for random time
			log.Printf("Waiting for random time")
			time.Sleep(15 * time.Second)
			log.Printf("Finish Waiting for random time")

			// Migrate the container
			req := &pb.PullRequest{DestinationAddr: *src, ContainerName: alias}
			conn, err := grpc.Dial(*dst, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(200*1024*1024),
			))
			if err != nil {
				fmt.Printf("did not connect: %v\n", err)
			}
			defer conn.Close()

			client := pb.NewPullContainerClient(conn)

			// The PullContainer will trigger the recordP on src
			res, err := client.PullContainer(context.Background(), req)
			if err != nil {
				fmt.Printf("Container migration failed: %v\n", err)
			}
			if res.Success {
				fmt.Printf("New container restored on %s with ID: %s\n", *dst, res.ContainerId)

				BytesMigrateCheckpoint := res.BytesMigrateCheckpoint
				BytesMigrateImage := res.BytesMigrateImage
				BytesMigrateVolume := res.BytesMigrateVolume
				
				fmt.Printf("BytesMigrateCheckpoint for migrating %s: %d \n", alias,BytesMigrateCheckpoint)
				fmt.Printf("BytesMigrateImage for migrating %s: %d \n", alias,BytesMigrateImage)
				fmt.Printf("BytesMigrateVolume for migrating %s: %d \n", alias,BytesMigrateVolume)
				err := recordMigrationData(csvFilePath, alias, i+1, BytesMigrateCheckpoint, BytesMigrateImage, BytesMigrateVolume)
				if err != nil {
					log.Printf("Failed to record migration data: %v", err)
				} else {
					log.Printf("Migration data recorded successfully for alias: %s, iteration: %d", alias, i+1)
			
					}	}
		}
	}
}


var containeralias = map[string]string{
	"192.168.116.150:5000/cnn:esgz":      "cnn",
	"192.168.116.150:5000/node:esgz":     "node",
	"192.168.116.150:5000/postgres:esgz": "postgres",
}

var containerCommands = map[string][]string{
	"192.168.116.150:5000/node:esgz":     {},
	"192.168.116.150:5000/cnn:esgz":      {"python3", "-u", "main.py", "--batch-size", "64", "--test-batch-size", "1000", "--epochs", "10", "--lr", "0.1", "--gamma", "0.7", "--log-interval", "1", "--save-model"},
	"192.168.116.150:5000/postgres:esgz": {},
}

var containerList = []string{
	"192.168.116.150:5000/cnn:esgz",
	"192.168.116.150:5000/node:esgz",
	"192.168.116.150:5000/postgres:esgz",
}

func recordMigrationData(filePath, alias string, iteration int, bytesCheckpoint, bytesImage, bytesVolume int64) error {
	// Check if the file exists
	fileExists := true
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fileExists = false
	}

	// Open the file in append mode, create it if it doesn't exist
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// If the file does not exist or is empty, write the header
	if !fileExists {
		header := []string{"Time", "Alias", "Iteration", "BytesMigrateCheckpoint", "BytesMigrateImage", "BytesMigrateVolume"}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write header: %v", err)
		}
	} else {
		// Check if the existing file is empty
		fileInfo, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat file: %v", err)
		}
		if fileInfo.Size() == 0 {
			header := []string{"Time", "Alias", "Iteration", "BytesMigrateCheckpoint", "BytesMigrateImage", "BytesMigrateVolume"}
			if err := writer.Write(header); err != nil {
				return fmt.Errorf("failed to write header: %v", err)
			}
		}
	}

	// Prepare the record to be written
	record := []string{
		time.Now().Format(time.RFC3339),
		alias,
		fmt.Sprintf("%d", iteration),
		fmt.Sprintf("%d", bytesCheckpoint),
		fmt.Sprintf("%d", bytesImage),
		fmt.Sprintf("%d", bytesVolume),
	}

	// Write the record to the CSV
	if err := writer.Write(record); err != nil {
		return fmt.Errorf("failed to write record: %v", err)
	}

	return nil
}
