package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"

	exp "github.com/zhiyuanGH/container-joint-migration/exputils"
	pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
	"google.golang.org/grpc"
)

// Define a struct that maps your JSON config
type Config struct {
	ImageFlags        map[string][]string `json:"imageFlags"`
	ContainerAlias    map[string]string   `json:"containerAlias"`
	ContainerCommands map[string][]string `json:"containerCommands"`
	ContainerList     []string            `json:"containerList"`
	Iteration         int                 `json:"iteration"`
	BandwidthLimit    []int               `json:"bandwidth"`
}

func main() {
	// 1. Load flags
	src := flag.String("src", "192.168.116.148:50051", "Server address for source host")
	dst := flag.String("dst", "192.168.116.149:50051", "Server address for destination host")
	registryAddr := flag.String("registry", "192.168.116.150:50051", "Server address for registry host")
	csvFilePath := flag.String("csv", "/home/base/code/box/data_t/dataCurrnet.csv", "Path to CSV output file")
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
				// Reset the source side
				exp.ResetOverlay()

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

				// Start container on src
				args := append([]string{"docker", "run", "-d", "--name", alias}, imageSpecificFlags...)
				args = append(args, imageName)
				args = append(args, commandArgs...)

				log.Printf("Executing: sudo %v\n", args)
				if _, _, err := executor.Execute(args); err != nil {
					log.Printf("Error during 'docker run': %v", err)
					continue
				}

				// Sleep for a random time
				log.Printf("Waiting for random time...")
				randomTime := time.Duration(rand.Intn(30)) * time.Second
				time.Sleep(randomTime)

				log.Printf("Finish Waiting.")

				// Migrate the container
				req := &pb.PullRequest{DestinationAddr: *src, ContainerName: alias}

				client := pb.NewPullContainerClient(conn)

				res, err := client.PullContainer(context.Background(), req)
				if err != nil {
					log.Printf("Container migration failed: %v\n", err)
					continue
				}

				if res.Success {
					// Logging logic
					log.Printf("New container restored on %s with ID: %s\n", *dst, res.ContainerId)
					recordReq := &pb.RecordRequest{
						ContainerName:  alias,
						RecordFileName: "",
					}

					recordClient := pb.NewRecordFClient(conn)
					if _, err := recordClient.RecordFReset(context.Background(), recordReq); err != nil {
						log.Printf("Record F failed: %v\n", err)
					}

					BytesMigrateCheckpoint := res.BytesMigrateCheckpoint
					BytesMigrateImage := res.BytesMigrateImage
					BytesMigrateVolume := res.BytesMigrateVolume

					secondsMigrateImage := res.SecondsMigrateImage.AsDuration().Milliseconds()
					secondsMigrateCheckpoint := res.SecondsMigrateCheckpoint.AsDuration().Milliseconds()
					secondsMigrateVolume := res.SecondsMigrateVolume.AsDuration().Milliseconds()

					log.Printf("BytesMigrateCheckpoint for %s: %d", alias, BytesMigrateCheckpoint)
					log.Printf("BytesMigrateImage for %s: %d", alias, BytesMigrateImage)
					log.Printf("BytesMigrateVolume for %s: %d", alias, BytesMigrateVolume)
					log.Printf("SecondsMigrateCheckpoint for %s: %d", alias, secondsMigrateCheckpoint)
					log.Printf("SecondsMigrateImage for %s: %d", alias, secondsMigrateImage)
					log.Printf("SecondsMigrateVolume for %s: %d", alias, secondsMigrateVolume)

					if err := recordMigrationData(
						*csvFilePath,
						alias,
						i+1,
						randomTime.Milliseconds(),
						BytesMigrateCheckpoint,
						BytesMigrateImage,
						BytesMigrateVolume,
						secondsMigrateCheckpoint,
						secondsMigrateImage,
						secondsMigrateVolume,
					); err != nil {
						log.Printf("Failed to record migration data: %v", err)
					} else {
						log.Printf("Migration data recorded successfully for alias: %s, iteration: %d", alias, i+1)
					}
				}
			}
		}
	}
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

// recordMigrationData is unchanged from your example
func recordMigrationData(
	filePath, alias string,
	iteration int,
	migrateWhen, bytesCheckpoint, bytesImage, bytesVolume int64,
	secondsCheckpoint, secondsImage, secondsVolume int64,
) error {
	// your existing CSV logic...
	header := []string{
		"Time", "Alias", "Iteration", "MigrateWhen",
		"BytesMigrateCheckpoint", "BytesMigrateImage", "BytesMigrateVolume",
		"MillisecondsMigrateCheckpoint", "MillisecondsMigrateImage", "MillisecondsMigrateVolume",
	}
	fileExists := true
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fileExists = false
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if !fileExists {

		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write header: %v", err)
		}
	} else {
		fileInfo, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat file: %v", err)
		}
		if fileInfo.Size() == 0 {

			if err := writer.Write(header); err != nil {
				return fmt.Errorf("failed to write header: %v", err)
			}
		}
	}

	record := []string{
		time.Now().Format(time.RFC3339),
		alias,
		fmt.Sprintf("%d", iteration),
		fmt.Sprintf("%d", migrateWhen),
		fmt.Sprintf("%d", bytesCheckpoint),
		fmt.Sprintf("%d", bytesImage),
		fmt.Sprintf("%d", bytesVolume),
		fmt.Sprintf("%d", secondsCheckpoint),
		fmt.Sprintf("%d", secondsImage),
		fmt.Sprintf("%d", secondsVolume),
	}

	if err := writer.Write(record); err != nil {
		return fmt.Errorf("failed to write record: %v", err)
	}

	return nil
}
