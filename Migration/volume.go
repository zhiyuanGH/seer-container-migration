package Migration

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
	"github.com/zhiyuanGH/container-joint-migration/utils"
)

func Createvolume(volres *pb.VolumeResponse) (binds string, err error) {
	if volres.NfsSource != "" {
		return createVolumeFromNFS(volres)
	}
	return createVolumeFromData(volres)
}

// volumeName is
func createVolumeFromNFS(volres *pb.VolumeResponse) (binds string, err error) {
	volumeName := volres.VolumeName
	nfsSource := volres.NfsSource

	source, err := utils.GetMountSource(volumeName)

	if err != nil {
		fmt.Printf("Cannot find mount on the destination host, so dont need to delete it: %v\n", err)
	}

	// Check if the directory is already mounted
	if source != "" {
		// Unmount the directory
		umountCmd := exec.Command("sudo", "umount", volumeName)
		if err := umountCmd.Run(); err != nil {
			return "", fmt.Errorf("failed to unmount directory %s: %w", volumeName, err)
		}
	}

	// Check if the directory exists
	if _, err := os.Stat(volumeName); !os.IsNotExist(err) {
		// Remove the directory
		removeCmd := exec.Command("sudo", "rm", "-rf", volumeName)
		if err := removeCmd.Run(); err != nil {
			return "", fmt.Errorf("failed to remove directory %s: %w", volumeName, err)
		}
	}

	// Create the directory with sudo
	mkdirCmd := exec.Command("sudo", "mkdir", volumeName)
	if err := mkdirCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", volumeName, err)
	}

	// Execute the mount command
	mountCmd := exec.Command("sudo", "mount", "-t", "nfs", nfsSource, volumeName)
	if err := mountCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to mount NFS: %w", err)
	}

	return fmt.Sprintf("%s:%s", volres.VolumeName, volres.Destination), nil
}

func createVolumeFromData(volres *pb.VolumeResponse) (binds string, err error) {
	volumeName := volres.VolumeName
	volumeData := volres.VolumeData

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}

	_, err = cli.VolumeCreate(context.Background(), volume.CreateOptions{
		Name: volumeName,
	})
	if err != nil {
		return "", err
	}

	volumeDir := fmt.Sprintf("/var/lib/docker/volumes/%s/_data", volumeName)
	os.MkdirAll(volumeDir, os.ModePerm)

	buf := bytes.NewBuffer(volumeData)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return "", err
	}
	tarReader := tar.NewReader(gz)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		target := filepath.Join(volumeDir, hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				return "", err
			}
		} else {
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(f, tarReader); err != nil {
				return "", err
			}
			f.Close()
		}
	}

	return fmt.Sprintf("%s:/%s", volumeName, volres.Destination), nil
}
