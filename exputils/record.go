package exputils

import (
	"log"
	"os"
)

// CommandExecutor is an interface that defines the Execute method

func CollectP(filename string) {
	logFile, err := os.OpenFile("/home/base/code/box/data_p/datacollectP.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		// If we can't open the log file, log to stderr and exit
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	err = RenameRecordFile(filename)
	if err != nil {
		log.Printf("Failed to rename file: %v", err)
	}
}

// filename must be the full path of the file, if it is a P file, it should be /home/base/code/box/data_p/xxx.csv
func RenameRecordFile(filename string) error {
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
