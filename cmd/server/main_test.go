package main

import (
	"testing"
	
)

func TestGetMountSource(t *testing.T) {
	mountPoint := "/mnt/nfs_share"
	expectedSource := "/dev/sda1"

	source, err := getMountSource(mountPoint)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if source != expectedSource {
		t.Errorf("got %s, want %s", source, expectedSource)
	}
}
 func TestXxx(t *testing.T) {
	
 }