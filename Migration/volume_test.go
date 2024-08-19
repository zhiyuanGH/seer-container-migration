package Migration

import (
	
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
)

// Mock the exec.Command to simulate the mount command
var execCommand = exec.Command

func TestCreateVolumeFromNFS(t *testing.T) {
	volRes := &pb.VolumeResponse{
		VolumeName:  "/mnt/nfs_share",
		NfsSource:   "192.168.116.148:/srv/nfs/share",
		Destination: "/data",
	}

	// Mock the exec.Command to simulate the mount command
	execCommand = func(name string, arg ...string) *exec.Cmd {
		cmd := exec.Command("echo", "mock mount command")
		return cmd
	}

	// Replace exec.Command with the mocked version
	defer func() { execCommand = exec.Command }()

	binds, err := createVolumeFromNFS(volRes)
	assert.NoError(t, err)
	assert.Equal(t, "nfs.example.com:/path/to/source:/data", binds)
}
