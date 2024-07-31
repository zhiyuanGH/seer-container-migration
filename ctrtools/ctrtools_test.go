package ctrtools

import (
    "os/exec"
    "strings"
    "testing"
)

// Helper function to run shell commands with sudo
func runShellCommand_test(t *testing.T, command string) string {
    cmd := exec.Command("bash", "-c", "echo 'gh' | sudo -S "+command)
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("failed to run command %s: %v, output: %s", command, err, output)
    }
    return strings.TrimSpace(string(output))
}

func TestResetSnapshotters(t *testing.T) {
    // Check Docker status before reset
    containersBefore := runShellCommand_test(t, "docker ps -a -q")
    if containersBefore == "" {
        t.Log("No containers running before reset")
    } else {
        t.Logf("Containers before reset: %s", containersBefore)
    }

    err := ResetSnapshotters()
    if err != nil {
        t.Fatalf("ResetSnapshotters failed: %v", err)
    }

    // Check Docker status after reset
    containersAfter := runShellCommand_test(t, "docker ps -a -q")
    if containersAfter != "" {
        t.Fatalf("Expected no containers after reset, but found: %s", containersAfter)
    }

    // Add additional checks as necessary, e.g., check if services are running, directories are cleaned up, etc.
}
