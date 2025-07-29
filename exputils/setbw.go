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

    // Define the tc commands to set the bandwidth limit and add latency
    commands := [][]string{
        // 1) Delete any existing root qdisc
        {"tc", "qdisc", "del", "dev", interfaceName, "root"},

        // 2) Add HTB root qdisc
        {"tc", "qdisc", "add", "dev", interfaceName, "root", "handle", "1:", "htb", "default", "30"},

        // 3) Create class 1:1 limited to ‘bw’ megabits
        {"tc", "class", "add", "dev", interfaceName,
            "parent", "1:", "classid", "1:1", "htb",
            "rate", fmt.Sprintf("%dmbit", bw)},

        // 4) Under that class, attach netem to add 100 ms delay (adds 100 ms one‑way;
        //    so packets see roughly 200 ms RTT unless you also shape the reverse)
        {"tc", "qdisc", "add", "dev", interfaceName,
            "parent", "1:1", "handle", "10:", "netem",
            "delay", "100ms"},

        // 5) Filter traffic to your target IP into the shaped+delayed class
        {"tc", "filter", "add", "dev", interfaceName,
            "protocol", "ip", "parent", "1:0", "prio", "1",
            "u32", "match", "ip", "dst", targetIP,
            "flowid", "1:1"},
    }

    for _, args := range commands {
        log.Printf("Executing: sudo %v\n", args)
        stdout, stderr, err := executor.Execute(args)
        if err != nil {
            log.Printf("Command failed: %v\nstdout: %s\nstderr: %s",
                err, stdout, stderr)
        }
        time.Sleep(1 * time.Second)
    }

    log.Println("Bandwidth and latency configured successfully.")
    return nil
}
