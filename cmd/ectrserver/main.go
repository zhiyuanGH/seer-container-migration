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
	exp "github.com/zhiyuanGH/container-joint-migration/exputils"
	pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedContainerMigrationServer
	pb.UnimplementedPullContainerServer
	pb.UnimplementedRecordFServer
}

func (s *server) PullContainer(ctx context.Context, req *pb.PullRequest) (*pb.PullResponse, error) {

	fmt.Printf("Received request to pull container from: %s\n", req.DestinationAddr)
	addr := req.DestinationAddr
	containerName := req.ContainerName
	newContainerID, err := Migration.PullContainerToLocalhost(addr, containerName, req.RecordFileName)
	if err != nil {
		log.Fatalf("Container migration failed: %v", err)
		return &pb.PullResponse{ContainerId: containerName, Success: false}, err
	}
	fmt.Printf("New container restored with ID: %s\n", newContainerID) // revise to log
	return &pb.PullResponse{ContainerId: newContainerID, Success: true}, nil
}

// this service is running on the dst side and record the f and reset the dst
func (s *server) RecordFReset(ctx context.Context, req *pb.RecordRequest) (*pb.RecordResponse, error) {
	fmt.Println("Wait for the container to run: ", req.ContainerName)
	if err := exp.Wait(req.ContainerName); err != nil {
		return &pb.RecordResponse{Success: false}, err
	}
	fmt.Println("Container exited, recording F: ", req.RecordFileName)
	if err := exp.RenameRecordFile(req.RecordFileName); err != nil {
		fmt.Printf("Error renaming record file F on dst: %v", err)
		return &pb.RecordResponse{Success: false}, err
	}
	exp.Reset()
	return &pb.RecordResponse{Success: true}, nil
}

func (s *server) CheckpointContainer(ctx context.Context, req *pb.CheckpointRequest) (*pb.CheckpointResponse, error) {

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %v", err)
	}

	// Inspect the container to get the full ID
	containerInfo, err := cli.ContainerInspect(ctx, req.ContainerId)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container with ID %s: %v", req.ContainerId, err)
	}
	fullContainerID := containerInfo.ID

	// Create a unique checkpoint ID
	checkpointID := fmt.Sprintf("checkpoint_%d", time.Now().Unix())
	fmt.Println("Creating checkpoint for container:", req.ContainerId, "with ID: ", checkpointID)

	// Create checkpoint
	if err := cli.CheckpointCreate(ctx, req.ContainerId, checkpoint.CreateOptions{CheckpointID: checkpointID, Exit: true}); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint for container %s: %v", req.ContainerId, err)
	}

	// Set checkpoint directory path
	checkpointDir := fmt.Sprintf("/var/lib/docker/containers/%s/checkpoints/%s", fullContainerID, checkpointID)

	// Initialize buffer for checkpoint data
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gz)

	// Walk through checkpoint directory and add files to the tar archive
	err = filepath.Walk(checkpointDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk checkpoint directory %s: %v", checkpointDir, err)
		}
		hdr, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return fmt.Errorf("failed to create tar header for file %s: %v", file, err)
		}
		hdr.Name = filepath.ToSlash(file[len(checkpointDir):])
		if err := tarWriter.WriteHeader(hdr); err != nil {
			return fmt.Errorf("failed to write header for file %s to tar archive: %v", file, err)
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("failed to open file %s for tar archiving: %v", file, err)
		}
		defer f.Close()
		if _, err := io.Copy(tarWriter, f); err != nil {
			return fmt.Errorf("failed to copy contents of file %s into tar archive: %v", file, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Close tar and gzip writers
	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %v", err)
	}

	// Handle renaming the record file if provided
	if req.RecordFileName != "" {
		fmt.Println("Renaming the filename of the record file: ", req.RecordFileName)
		if err := exp.RenameRecordFile(req.RecordFileName); err != nil {
			fmt.Printf("Error renaming record file P on src: %v", err)
			return nil, fmt.Errorf("failed to rename record file %s: %v", req.RecordFileName, err)
		}
	} else {
		fmt.Println("No record file to rename")
	}

	// Return checkpoint response with the checkpoint data
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
		return &pb.VolumeResponse{}, nil
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
		grpc.UnaryInterceptor(UnaryTrafficInterceptor),
	)

	// Register both services
	pb.RegisterContainerMigrationServer(grpcServer, &server{})
	pb.RegisterPullContainerServer(grpcServer, &server{}) // Register PullContainer service
	pb.RegisterRecordFServer(grpcServer, &server{})       // Register RecordF service

	log.Printf("Server listening at %v", lis.Addr())
	fmt.Println("Testing")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func UnaryTrafficInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// Log incoming request size
	reqSize := getSize(req)
	log.Printf("Incoming Request - Method:%s Size:%d bytes", info.FullMethod, reqSize)

	// Handle the request
	resp, err := handler(ctx, req)

	// Log outgoing response size
	respSize := getSize(resp)
	log.Printf("Outgoing Response - Method:%s Size:%d bytes", info.FullMethod, respSize)

	return resp, err
}

// getSize calculates the approximate size of the message in bytes.
func getSize(msg interface{}) int {
	if msg == nil {
		return 0
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return 0
	}
	return len(data)
}