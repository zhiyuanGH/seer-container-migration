package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"

	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/checkpoint"
	"github.com/docker/docker/client"
	"github.com/zhiyuanGH/container-joint-migration/Migration"
	pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedContainerMigrationServer
	pb.UnimplementedPullContainerServer
}

func (s *server) PullContainer(ctx context.Context, req *pb.PullRequest) (*pb.PullResponse, error) {

	fmt.Printf("Received request to pull container from: %s\n", req.DestinationAddr)
	addr := req.DestinationAddr
	containerName := req.ContainerName

	newContainerID, err := Migration.PullContainerToLocalhost(addr, containerName)
	if err != nil {
		log.Fatalf("Container migration failed: %v", err)
		return &pb.PullResponse{ContainerId: containerName, Success: false}, err
	}

	fmt.Printf("New container restored with ID: %s\n", newContainerID) // revise to log 
	return &pb.PullResponse{ContainerId: newContainerID, Success: true}, nil
}

func (s *server) CheckpointContainer(ctx context.Context, req *pb.CheckpointRequest) (*pb.CheckpointResponse, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	// Inspect the container to get the full ID
	containerInfo, err := cli.ContainerInspect(ctx, req.ContainerId)
	if err != nil {
		return nil, err
	}
	fullContainerID := containerInfo.ID

	checkpointID := fmt.Sprintf("checkpoint_%d", time.Now().Unix())
	if err := cli.CheckpointCreate(ctx, req.ContainerId, checkpoint.CreateOptions{CheckpointID: checkpointID, Exit: true}); err != nil {
		return nil, err
	}

	checkpointDir := fmt.Sprintf("/var/lib/docker/containers/%s/checkpoints/%s", fullContainerID, checkpointID)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gz)

	err = filepath.Walk(checkpointDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(file[len(checkpointDir):])
		if err := tarWriter.WriteHeader(hdr); err != nil {
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(tarWriter, f); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	return &pb.CheckpointResponse{CheckpointId: checkpointID, CheckpointData: buf.Bytes()}, nil
}

func (s *server) TransferContainerInfo(ctx context.Context, req *pb.ContainerInfoRequest) (*pb.ContainerInfoResponse, error) {
	
	fmt.Printf("Received request to migrate container: %s\n", req.ContainerId)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	containerInfo, err := cli.ContainerInspect(ctx, req.ContainerId)
	if err != nil {
		return nil, err
	}

	// Marshal containerInfo into JSON
	containerInfoJSON, err := json.Marshal(containerInfo)
	if err != nil {
		return nil, err
	}

	return &pb.ContainerInfoResponse{ContainerInfo: containerInfoJSON}, nil
}

func (s *server) TransferVolume(ctx context.Context, req *pb.VolumeRequest) (*pb.VolumeResponse, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	containerID := req.ContainerId
	containerInfo, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, err
	}

	var volumeName string //volumeName is applicable for both local volume and nfs bind mount, but just has different names
	var nfsSource string  //nfsSource is only assigned if the container has a nfs bind mount

	if len(containerInfo.Mounts) == 0 {
		return nil, fmt.Errorf("no mounts found for container: %s", containerID)
	}
	var destination string

	for _, mount := range containerInfo.Mounts {
		destination = mount.Destination // assign the value to destination
		if mount.Type == "volume" {
			volumeName = mount.Name
			break
		}
		if mount.Type == "bind" {
			volumeName = mount.Source
			source, err := getMountSource(mount.Source)
			if err != nil {
				return nil, err
			}
			nfsSource = source
			break
		}

	}

	// If the container has a local volume, transfer the volume data
	if nfsSource == "" {
		volume, err := cli.VolumeInspect(ctx, volumeName)
		if err != nil {
			return nil, err
		}

		volumeDir := volume.Mountpoint
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tarWriter := tar.NewWriter(gz)

		err = filepath.Walk(volumeDir, func(file string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			hdr, err := tar.FileInfoHeader(fi, file)
			if err != nil {
				return err
			}
			hdr.Name = filepath.ToSlash(file[len(volumeDir):])
			if err := tarWriter.WriteHeader(hdr); err != nil {
				return err
			}
			if !fi.Mode().IsRegular() {
				return nil
			}
			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tarWriter, f); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		if err := tarWriter.Close(); err != nil {
			return nil, err
		}
		if err := gz.Close(); err != nil {
			return nil, err
		}

		return &pb.VolumeResponse{VolumeName: volumeName, VolumeData: buf.Bytes(), Destination: destination}, nil
	}

	// If the container has a nfs bind mount, return the NFS source.
	return &pb.VolumeResponse{VolumeName: volumeName, NfsSource: nfsSource, Destination: destination}, nil

}

func getMountSource(mountPoint string) (string, error) {
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

func main() {
	lis, err := net.Listen("tcp4", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(200 * 1024 * 1024),
	)
	
	// Register both services
	pb.RegisterContainerMigrationServer(grpcServer, &server{})
	pb.RegisterPullContainerServer(grpcServer, &server{}) // Register PullContainer service

	log.Printf("Server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}