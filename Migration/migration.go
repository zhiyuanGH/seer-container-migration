package Migration

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
	"google.golang.org/grpc"
)

// PullImageIfNotExists pulls the specified image if it does not exist locally
func PullImageIfNotExists(cli *client.Client, imageName string) error {
	_, _, err := cli.ImageInspectWithRaw(context.Background(), imageName)
	if err != nil {
		fmt.Printf("Image %s not found locally. Pulling...\n", imageName)
		reader, err := cli.ImagePull(context.Background(), imageName, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("could not pull image: %v", err)
		}
		defer reader.Close()
		io.Copy(os.Stdout, reader)
	}
	return nil
}

func restoreContainer(checkpointData []byte, image string, name string, binds string) (string, error) {
	fmt.Printf("Starting restore container %s with image %s\n", name, image)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("error creating Docker client: %v", err)
	}

	// Pull the image if it doesn't exist
	err = PullImageIfNotExists(cli, image)
	if err != nil {
		return "", fmt.Errorf("error pulling image: %v", err)
	}
	fmt.Printf("Pulled image %s successfully \n", image)

	// Handle binds: If binds is empty, pass nil to HostConfig.Binds
	var bindList []string
	if binds != "" {
		bindList = append(bindList, binds)
	}

	// Create the container
	newResp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: image,
		Cmd:   []string{"sh", "-c", "i=0; while true; do echo $i; i=$((i+1)); sleep 1; done"},
		Tty:   false,
	}, &container.HostConfig{
		Binds: bindList, // Use bindList which is nil if binds was empty
	}, nil, nil, name)
	if err != nil {
		return "", fmt.Errorf("error creating container: %v", err)
	}
	fmt.Printf("Created container with ID: %s and Name: %s\n", newResp.ID, name)

	// Create checkpoint directory
	checkpointDir := fmt.Sprintf("/var/lib/docker/containers/%s/checkpoints/checkpoint1", newResp.ID)
	err = os.MkdirAll(checkpointDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("error creating checkpoint directory: %v", err)
	}
	fmt.Print("Created checkpoint directory successfully\n")

	// Unzip the checkpoint data
	buf := bytes.NewBuffer(checkpointData)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return "", fmt.Errorf("error creating gzip reader for checkpoint data: %v", err)
	}
	tarReader := tar.NewReader(gz)

	// Extract the checkpoint data
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("error reading tar header: %v", err)
		}

		target := filepath.Join(checkpointDir, hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			err = os.MkdirAll(target, os.ModePerm)
			if err != nil {
				return "", fmt.Errorf("error creating directory in checkpoint: %v", err)
			}
		} else {
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return "", fmt.Errorf("error opening file in checkpoint: %v", err)
			}
			_, err = io.Copy(f, tarReader)
			if err != nil {
				return "", fmt.Errorf("error copying data to file in checkpoint: %v", err)
			}
			f.Close()
		}
	}
	fmt.Println("Extracted checkpoint data successfully")

	// Start the container with the checkpoint
	err = cli.ContainerStart(context.Background(), newResp.ID, container.StartOptions{CheckpointID: "checkpoint1"})
	if err != nil {
		return "", fmt.Errorf("error starting container: %v", err)
	}
	fmt.Printf("Container started successfully with ID: %s\n", newResp.ID)

	return newResp.ID, nil
}


// currently PullContainerToLocalhost is more like to fetch a container from given address to local host
func PullContainerToLocalhost(addr string, containerID string, recordfilename string) (string, error) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(200*1024*1024),
	))

	if err != nil {
		return "", fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()

	grpcClient := pb.NewContainerMigrationClient(conn)

	startTime := time.Now()

	infoReq := &pb.ContainerInfoRequest{ContainerId: containerID}
	infoRes, err := grpcClient.TransferContainerInfo(context.Background(), infoReq)
	if err != nil {
		return "", fmt.Errorf("could not get container info: %v", err)
	}
	var containerInfo types.ContainerJSON
	err = json.Unmarshal(infoRes.ContainerInfo, &containerInfo)
	if err != nil {
		return "", fmt.Errorf("could not unmarshal container info: %v", err)
	}
	fmt.Printf("Container Name: %s\n", containerInfo.Name)
	fmt.Printf("Container Image: %s\n", containerInfo.Config.Image)
	fmt.Printf("Container State: %s\n", containerInfo.State.Status)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("error creating Docker client: %v", err)
	}

	err = PullImageIfNotExists(cli, containerInfo.Config.Image)
	if err != nil {
		return "", fmt.Errorf("error pulling image: %v", err)
	}
	fmt.Printf("Pulled image %s successfully \n", containerInfo.Config.Image)

	

	volReq := &pb.VolumeRequest{ContainerId: containerID}
	volRes, err := grpcClient.TransferVolume(context.Background(), volReq)
	if err != nil {
		return "", fmt.Errorf("could not transfer volume: %v", err)
	}
	fmt.Printf("got volume res \n")

	binds, volCreateErr := Createvolume(volRes)
	if volCreateErr != nil {
		return "", fmt.Errorf("could not create volume: %v", volCreateErr)
	}

	req := &pb.CheckpointRequest{ContainerId: containerID, RecordFileName: recordfilename}
	res, err := grpcClient.CheckpointContainer(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("could not checkpoint container: %v", err)
	}
	fmt.Print("got checkpoint res \n")

	newContainerID, err := restoreContainer(res.CheckpointData, containerInfo.Config.Image, containerInfo.Name, binds)
	if err != nil {
		return "", fmt.Errorf("could not restore container: %v", err)
	}

	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	fmt.Printf("Time taken from checkpointing container to finishing restore: %s\n", elapsedTime)
	return newContainerID, nil
}
