package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

//currently two places use this function: 
//1. when checkpointing a container, we need to get the nfsSource
//2. when restoring a container, we need to check whether already have the volume dir, if so, we need to delete it, but we need to unmount it first to avoid affecting the source

func GetMountSource(mountPoint string) (string, error) {
	// Execute findmnt command
	cmd := exec.Command("findmnt", "--output", "SOURCE", "--noheadings", mountPoint)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run findmnt: %v", err)
	}

	// Get the output and trim any extra whitespace
	source := strings.TrimSpace(out.String())

	// If no source is found, return an error
	if source == "" {
		return "", fmt.Errorf("no source found for mount point: %s", mountPoint)
	}

	return source, nil
}
