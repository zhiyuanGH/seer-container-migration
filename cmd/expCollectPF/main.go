package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	exp "github.com/zhiyuanGH/container-joint-migration/exputils"
	pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
	"google.golang.org/grpc"
)

type Config struct {
	ImageFlags        map[string][]string `json:"imageFlags"`
	ContainerAlias    map[string]string   `json:"containerAlias"`
	ContainerCommands map[string][]string `json:"containerCommands"`
	ContainerList     []string            `json:"containerList"`
	Iteration         int                 `json:"iteration"`
	BandwidthLimit    []int               `json:"bandwidth"`
}

// loadConfig reads the JSON config file and unmarshals it into Config
func loadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return &cfg, nil
}

func main() {
	// 1. Load flags
	src := flag.String("src", "192.168.116.148:50051", "Server address for source host")
	dst := flag.String("dst", "192.168.116.149:50051", "Server address for destination host")
	registryAddr := flag.String("registry", "192.168.116.150:50051", "Server address for registry host")
	configPath := flag.String("config", "/home/base/code/container-joint-migration/config.json", "Path to the JSON config file")

	flag.Parse()

	conn, err := grpc.Dial(*dst, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(200*1024*1024),
	))
	if err != nil {
		log.Printf("Did not connect: %v\n", err)
	}
	defer conn.Close()

	registryconn, err := grpc.Dial(*registryAddr, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(200*1024*1024),
	))
	if err != nil {
		log.Printf("Did not connect: %v\n", err)
	}
	defer registryconn.Close()

	// 2. Parse config file
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 3. Begin main logic
	executor := &exp.RealCommandExecutor{}
	// Migrate the container using the provided or default server address and container ID
	for _, bw := range cfg.BandwidthLimit {
		// Set bandwidth limit
		log.Printf("Setting bandwidth limit to %d Mbit/s\n", bw)
		bwreq := &pb.BandwidthLimitRequest{BandwidthLimit: int64(bw)}
		beClientRegistry := pb.NewSetBandwidthLimitClient(registryconn)
		if _, err := beClientRegistry.SetBandwidthLimit(context.Background(), bwreq); err != nil {
			log.Printf("Failed to set bandwidth limit on registry: %v", err)
			continue
		}
		if err := exp.SetBW(bw); err != nil {
			log.Printf("Failed to set bandwidth limit: %v", err)
			continue
		}
		//start iterate over the container list
		for _, imageName := range cfg.ContainerList {
			for i := 0; i < cfg.Iteration; i++ {
				// Grab everything from cfg
				commandArgs, okCmd := cfg.ContainerCommands[imageName]
				alias, okAlias := cfg.ContainerAlias[imageName]
				imageSpecificFlags, okFlags := cfg.ImageFlags[imageName]

				if !okCmd {
					commandArgs = []string{}
				}
				if !okAlias {
					alias = "container"
				}
				if !okFlags {
					imageSpecificFlags = []string{}
				}
				// Reset the src
				exp.ResetStargz()

				recordPFileName := fmt.Sprintf("/home/base/code/box/data_p/%s/%s_%d_%d.csv", alias, alias, i+1, bw)
				recordFFileName := fmt.Sprintf("/home/base/code/box/data_f/%s/%s_%d_%d.csv", alias, alias, i+1, bw)

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
				sleeptime := time.Duration(i+2) * time.Second
				log.Printf("Waiting for time: %v\n", sleeptime)
				time.Sleep(sleeptime)

				// Migrate the container
				req := &pb.PullRequest{DestinationAddr: *src, ContainerName: alias, RecordFileName: recordPFileName}
				client := pb.NewPullContainerClient(conn)
				// The PullContainer will trigger the recordP on src
				res, err := client.PullContainer(context.Background(), req)
				if err != nil {
					fmt.Printf("Container migration failed: %v\n", err)
				}
				if res.Success {
					fmt.Printf("New container restored on %s with ID: %s\n", *dst, res.ContainerId)
					// Record the F on dst
					recordReq := &pb.RecordRequest{ContainerName: alias, RecordFileName: recordFFileName}
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
	}
}
