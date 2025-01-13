package main

import (

	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"os"
	"time"

	exp "github.com/zhiyuanGH/container-joint-migration/exputils"

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

	csvFilePath := flag.String("csv", "/home/base/code/box/data_t/seer.csv", "Path to CSV output file")
	configPath := flag.String("config", "/home/base/code/container-joint-migration/config.json", "Path to the JSON config file")

	flag.Parse()

	// 2. Parse config file
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 3. Begin main logic
	executor := &exp.RealCommandExecutor{}
	for _, bw := range cfg.BandwidthLimit {
		//start iterate over the container list
		for _, imageName := range cfg.ContainerList {
			for i := 0; i < cfg.Iteration; i++ {
				// Reset the source side
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

				starttime := time.Now()
				fmt.Println("Start time: ", starttime)

				sleeptime := time.Duration(i+2) * time.Second
				sleeptime = time.Duration(2) * time.Second
				log.Printf("Waiting for time: %v\n", sleeptime)
				time.Sleep(sleeptime)
				log.Printf("Finish Waiting.")

				if true {
					// Logging logic
					timeoutDuration := 900 * time.Second

					if err := exp.Wait(alias, timeoutDuration); err != nil {
						log.Printf("Failed to wait for container: %v", err)
						continue
					}
					exp.ResetOverlay(false)
					endtime := time.Now()
					timeElapsed := endtime.Sub(starttime)
					fmt.Println("Time elapsed: ", timeElapsed)

					BytesMigrateCheckpoint := 0
					BytesMigrateImage := 0
					BytesMigrateVolume := 0
					secondsMigrateImage := 0
					secondsMigrateCheckpoint := 0
					secondsMigrateVolume := 0

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
						bw,
						sleeptime.Milliseconds(),

						int64(BytesMigrateCheckpoint),
						int64(BytesMigrateImage),
						int64(BytesMigrateVolume),
						int64(secondsMigrateCheckpoint),
						int64(secondsMigrateImage),
						int64(secondsMigrateVolume),
						timeElapsed.Milliseconds(),
					); err != nil {
						log.Printf("Failed to record migration data: %v", err)
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
	iteration, bw int,
	migrateWhen, bytesCheckpoint, bytesImage, bytesVolume int64,
	secondsCheckpoint, secondsImage, secondsVolume, containerFinishTime int64,
) error {
	// your existing CSV logic...
	header := []string{
		"Time", "Alias", "Iteration", "BandwidthLimit", "MigrateWhen",
		"BytesMigrateCheckpoint", "BytesMigrateImage", "BytesMigrateVolume",
		"MillisecondsMigrateCheckpoint", "MillisecondsMigrateImage", "MillisecondsMigrateVolume", "ContainerFinishTime",
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
		fmt.Sprintf("%d", bw),
		fmt.Sprintf("%d", migrateWhen),
		fmt.Sprintf("%d", bytesCheckpoint),
		fmt.Sprintf("%d", bytesImage),
		fmt.Sprintf("%d", bytesVolume),
		fmt.Sprintf("%d", secondsCheckpoint),
		fmt.Sprintf("%d", secondsImage),
		fmt.Sprintf("%d", secondsVolume),
		fmt.Sprintf("%d", containerFinishTime),
	}

	if err := writer.Write(record); err != nil {
		return fmt.Errorf("failed to write record: %v", err)
	}

	return nil
}
