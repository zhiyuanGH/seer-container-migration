package exputils

import (
	"fmt"
	"log"
	"time"
)

// Assume RealCommandExecutor is defined elsewhere with an Execute method.
// type RealCommandExecutor struct { /* fields */ }
//
// func (r *RealCommandExecutor) Execute(args []string) (stdout string, stderr string, err error) {
//     // Implementation to execute the command and return stdout, stderr, and error.
// }

func SetBW(bw int) error {
	executor := &RealCommandExecutor{}

	// Define the network interface and target IP
	interfaceName := "ens33"
	targetIP := "192.168.116.149"

	// Define the tc commands to set the bandwidth limit
	commands := [][]string{
		// Step 1: Delete existing qdisc on the interface
		{"tc", "qdisc", "del", "dev", interfaceName, "root"},

		// Step 2: Add root htb qdisc with handle 1: and default class 30
		{"tc", "qdisc", "add", "dev", interfaceName, "root", "handle", "1:", "htb", "default", "30"},

		// Step 3: Add class 1:1 with the specified rate
		{"tc", "class", "add", "dev", interfaceName, "parent", "1:", "classid", "1:1", "htb", "rate", fmt.Sprintf("%dmbit", bw)},

		// Step 4: Add filter to match traffic destined for targetIP and direct it to class 1:1
		{"tc", "filter", "add", "dev", interfaceName, "protocol", "ip", "parent", "1:0", "prio", "1", "u32", "match", "ip", "dst", targetIP, "flowid", "1:1"},
	}

	for _, args := range commands {
		// Log the command being executed
		log.Printf("Executing: sudo %v\n", args)

		// Execute the command using the executor
		stdout, stderr, err := executor.Execute(args)

		if err != nil {
			return fmt.Errorf("command failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
		}

		// Optional: Wait for a short duration between commands to ensure they execute properly
		time.Sleep(1 * time.Second)
	}

	log.Println("Bandwidth limit set successfully.")
	return nil
}
