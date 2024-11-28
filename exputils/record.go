package exputils

import (
	"log"

)

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
