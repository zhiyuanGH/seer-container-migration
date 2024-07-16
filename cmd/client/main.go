package main

import (
    "fmt"
    "log"

    "github.com/zhiyuanGH/container-joint-migration/ctrtools"
)

func main() {
    serverAddress := "source-host:50051"
    containerID := "source-container-id"

    newContainerID, err := ctrtools.MigrateContainer(serverAddress, containerID)
    if err != nil {
        log.Fatalf("Container migration failed: %v", err)
    }

    fmt.Printf("New container restored with ID: %s\n", newContainerID)
}
