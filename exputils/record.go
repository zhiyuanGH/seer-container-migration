package exputils

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// RenameRecordFile renames the file and creates the target directory if it does not exist
func RenameRecordFile(filename string) error {
	// Extract the directory path from the filename
	dir := strings.TrimSuffix(filename, "/" + strings.Split(filename, "/")[len(filename)-1])

	// Check if the directory exists, and create it if it doesn't
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Directory doesn't exist, so create it
		log.Printf("Directory does not exist. Creating directory: %s", dir)
		err := os.MkdirAll(dir, 0755) // 0755 permissions allow read, write, and execute for owner, and read & execute for others
		if err != nil {
			log.Printf("Error creating directory: %v", err)
			return fmt.Errorf("error creating directory: %v", err)
		}
	}

	// Proceed with renaming the file
	executor := &RealCommandExecutor{}
	renameArgs := []string{"mv", "/home/base/code/box/tmpFileAccess.csv", filename}
	log.Printf("Renaming file: sudo %v\n", renameArgs)
	_, _, err := executor.Execute(renameArgs)
	if err != nil {
		log.Printf("Failed to rename file: %v", err)
		return err
	}
	return nil
}
