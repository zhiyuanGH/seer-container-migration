package exputils

import (
	"fmt"
	"log"
	"time"

)

func Wait(container string, timeout time.Duration) error {
	executor := &RealCommandExecutor{}
	waitArgs := []string{"docker", "wait", container}
	killArgs := []string{"docker", "kill", container}
	log.Printf("Waiting for container to exit: %v\n", waitArgs)

	done := make(chan error, 1)

	// Start the docker wait command in a separate goroutine
	go func() {
		_, _, err := executor.Execute(waitArgs)
		if err != nil {
			log.Printf("Error during 'docker wait': %v", err)
			done <- fmt.Errorf("failed to wait for container %s: %w", container, err)
			return
		}
		done <- nil
	}()

	// Use a select statement to wait for either the wait command to finish or the timeout
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		log.Printf("Timeout reached (%v), killing container %s", timeout, container)
		_, _, killErr := executor.Execute(killArgs)
		if killErr != nil {
			log.Printf("Error killing container %s: %v", container, killErr)
			return fmt.Errorf("timeout waiting for container %s and failed to kill: %w", container, killErr)
		}
		return nil
	}
}