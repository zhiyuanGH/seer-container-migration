package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/zhiyuanGH/container-joint-migration/Migration"
)

func main() {
	// Define flags for server address and container ID with default values
	serverAddress := flag.String("server", "192.168.116.148:50051", "Server address for container migration")
	containerID := flag.String("container", "loooper2", "ID of the container to migrate")

	// Parse the flags
	flag.Parse()

	// Migrate the container using the provided or default server address and container ID
	newContainerID, err := Migration.MigrateContainerToLocalhost(*serverAddress, *containerID)
	if err != nil {
		log.Fatalf("Container migration failed: %v", err)
	}

	fmt.Printf("New container restored with ID: %s\n", newContainerID)
}
