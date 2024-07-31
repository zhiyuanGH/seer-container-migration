package main

import (
	"fmt"
	"log"

	"github.com/zhiyuanGH/container-joint-migration/ctrtools"
)

func main() {
	serverAddress := "192.168.116.148:50051"
	containerID := "loooper2"

	newContainerID, err := ctrtools.MigrateContainerToLocalhost(serverAddress, containerID)
	if err != nil {
		log.Fatalf("Container migration failed: %v", err)
	}

	fmt.Printf("New container restored with ID: %s\n", newContainerID)
}
