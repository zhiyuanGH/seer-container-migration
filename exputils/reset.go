package exputils

import (
	"log"
	"time"
)

func Reset() {
	executor := &RealCommandExecutor{}
	commands := [][]string{
		{"systemctl", "restart", "docker"},
		{"docker", "system", "prune", "-f"},
		{"systemctl", "restart", "docker"},
		{"sh", "-c", "rm -rf /var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/*"},
		{"systemctl", "restart", "containerd"},
	}
	commands_2 := [][]string{
		{"systemctl", "restart", "stargz-snapshotter"},
		{"systemctl", "stop", "stargz-snapshotter"},
		{"sh", "-c", "rm -rf /var/lib/containerd-stargz-grpc/*"},
		{"systemctl", "restart", "stargz-snapshotter"},
		{"ctr", "-n", "moby", "i", "prune", "--all"},
	}
	for _, args := range commands {
		// Execute the command using the executor
		log.Printf("Executing: sudo %v\n", args)
		stdout, stderr, err := executor.Execute(args)
		if err != nil {
			log.Printf("Command failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
			continue
		}
		time.Sleep(1 * time.Second)
	}
	for _, args := range commands_2 {
		// Execute the command using the executor
		log.Printf("Executing: sudo %v\n", args)
		_, _, err := executor.Execute(args)
		if err != nil {
			log.Println("Error:", err)
			continue
		}
		time.Sleep(1 * time.Second)
	}
	for _, args := range commands_2 {
		// Execute the command using the executor
		log.Printf("Executing: sudo %v\n", args)
		stdout, stderr, err := executor.Execute(args)
		if err != nil {
			log.Printf("Command failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
		}
		time.Sleep(1 * time.Second)
	}

	log.Println("Docker reset completed successfully.")
}
