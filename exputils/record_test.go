package exputils

import (
	"testing"
	"os"


	"github.com/stretchr/testify/assert"
)

func TestRenameRecordFile(t *testing.T) {
	// Test case: Successful file rename
	t.Run("Success", func(t *testing.T) {
		// Prepare the source and target filenames for renaming
		sourceFile := "/home/base/code/box/data_p/cnn_2.csv"  // Ensure this file exists before running the test
		targetFile := "/home/base/code/box/data_p/cnn_1.csv"

		// Create the source file for the test (make sure the source file exists)
		err := os.WriteFile(sourceFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}
		defer os.Remove(sourceFile) // Clean up the file after the test

		// Call RenameRecordFile (this should succeed)
		err = RenameRecordFile(targetFile)

		// Assert that there is no error
		assert.NoError(t, err)

		// Verify that the file was renamed by checking if the target file exists
		_, err = os.Stat(targetFile)
		assert.NoError(t, err, "Target file should exist after rename")
		
		// Optionally, verify that the source file no longer exists
		_, err = os.Stat(sourceFile)
		assert.True(t, os.IsNotExist(err), "Source file should no longer exist")
	})

	// Test case: Error due to non-existent source file
	t.Run("Error_NonExistentSourceFile", func(t *testing.T) {
		// Test with a non-existent source file

		targetFile := "/path/to/destination/file.csv"

		// Call RenameRecordFile and expect an error
		err := RenameRecordFile(targetFile)

		// Assert that there is an error
		assert.Error(t, err)

		// Assert that the error contains the correct message
		assert.Contains(t, err.Error(), "failed to rename")
	})

	// Test case: Error due to insufficient permissions (make sure to set up a file with no write permissions)
	t.Run("Error_InsufficientPermissions", func(t *testing.T) {
		// Create the source file but set no permissions
		sourceFile := "/path/to/existing/file.csv"  // Ensure this file exists before running the test
		targetFile := "/path/to/destination/file.csv"
		err := os.WriteFile(sourceFile, []byte("test content"), 0444)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}
		defer os.Remove(sourceFile) // Clean up the file after the test

		// Call RenameRecordFile and expect an error due to insufficient permissions
		err = RenameRecordFile(targetFile)

		// Assert that there is an error
		assert.Error(t, err)

		// Assert that the error contains the correct message
		assert.Contains(t, err.Error(), "failed to rename")
	})
}
