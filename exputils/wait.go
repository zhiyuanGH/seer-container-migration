package exputils

import (
	"fmt"
	"log"

)



func Wait(container string) error{
	executor := &RealCommandExecutor{}
	waitArgs := []string{"docker", "wait", container}
	log.Printf("Waiting for container to exit: sudo %v\n", waitArgs)
	_, _, err := executor.Execute( waitArgs)
	if err != nil {
		log.Printf("Error during 'docker wait': %v", err)
		return fmt.Errorf("failed to wait for container %s: %w", container, err)
	}
	return nil
}
