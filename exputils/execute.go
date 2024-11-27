package exputils
import (
	"bytes"

	"os/exec"
)
type CommandExecutor interface {
	Execute(password string, args []string) (stdout string, stderr string, err error)
}

// RealCommandExecutor implements CommandExecutor and executes real commands
type RealCommandExecutor struct{}

func (e *RealCommandExecutor) Execute(args []string) (string, string, error) {
	password := "gh"
	// Prepare the command with sudo -S
	cmdArgs := append([]string{"-S"}, args...)
	cmd := exec.Command("sudo", cmdArgs...)

	// Set up the buffers to capture stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Provide the password to sudo via stdin
	cmd.Stdin = bytes.NewBufferString(password + "\n")

	// Execute the command
	err := cmd.Run()
	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()
	return stdout, stderr, err
}
