package exputils

import (
	"log"
	"time"
)

func ResetOverlay(deleteImage ...bool) {
	deleteImageFlag := true

	if len(deleteImage) > 0 {
		deleteImageFlag = deleteImage[0]
	}
	executor := &RealCommandExecutor{}
	var commands [][]string
	if deleteImageFlag {
		commands = [][]string{
			{"systemctl", "restart", "docker"},
			{"docker", "system", "prune", "-af"},
			{"systemctl", "stop", "docker"},
			{"sh", "-c", "rm -rf /var/lib/docker/*"},
			{"systemctl", "restart", "docker"},
			{"systemctl", "restart", "containerd"},
			{"systemctl", "restart", "docker"},
		}
	} else {
		commands = [][]string{
			{"systemctl", "restart", "docker"},
			{"docker", "system", "prune", "-f"},
			{"systemctl", "restart", "containerd"},
			{"systemctl", "restart", "docker"},
		}
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
	log.Println("OverlayFS reset completed successfully.")
}

func ResetStargz(deleteImage ...bool) {
	deleteImageFlag := true
	if len(deleteImage) > 0 {
		deleteImageFlag = deleteImage[0]
	}
	executor := &RealCommandExecutor{}
	commands := [][]string{
		{"systemctl", "restart", "docker"},
		{"docker", "system", "prune", "-af"},
		{"systemctl", "stop", "docker"},
		{"sh", "-c", "rm -rf /var/lib/docker/*"},
		{"sh", "-c", "rm -rf /var/lib/containerd/*"},
		{"systemctl", "restart", "docker"},
		{"systemctl", "restart", "containerd"},
		{"systemctl", "restart", "docker"},
	}
	commands_2 := [][]string{
		{"systemctl", "restart", "stargz-snapshotter"},
		{"systemctl", "stop", "stargz-snapshotter"},
		{"sh", "-c", "rm -rf /var/lib/containerd-stargz-grpc/*"},
		{"systemctl", "restart", "stargz-snapshotter"},
	}
	commands_3 := [][]string{
		{"systemctl", "restart", "docker"},
	}

	if deleteImageFlag {

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
		for _, args := range commands_3 {
			// Execute the command using the executor
			log.Printf("Executing: sudo %v\n", args)
			stdout, stderr, err := executor.Execute(args)
			if err != nil {
				log.Printf("Command failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
			}
			time.Sleep(1 * time.Second)
		}
		log.Println("Stargz reset completed successfully.")
		return
	} else {
		commands = [][]string{
			{"systemctl", "restart", "docker"},
			{"docker", "system", "prune", "-f"},
			{"systemctl", "restart", "containerd"},
			{"systemctl", "restart", "docker"},
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
		log.Println("Stargz reset completed successfully.")
		return
	}
}
