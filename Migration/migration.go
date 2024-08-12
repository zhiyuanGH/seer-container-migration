package Migration

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
	"google.golang.org/grpc"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
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

func restoreContainer(checkpointData []byte, volumeName string) (string, error) {
	fmt.Println("Starting restoreContainer function")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("error creating Docker client: %v", err)
	}

	imageName := "ghcr.io/stargz-containers/golang:1.18-esgz"
	err = PullImageIfNotExists(cli, imageName)
	if err != nil {
		return "", fmt.Errorf("error pulling image: %v", err)
	}
	fmt.Println("Pulled image successfully")

	newResp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: imageName,
		Cmd:   []string{"sh", "-c", "i=0; while true; do echo $i; i=$((i+1)); sleep 1; done"},
		Tty:   false,
	}, &container.HostConfig{
		Binds: []string{fmt.Sprintf("%s:/data", volumeName)},
	}, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("error creating container: %v", err)
	}
	fmt.Printf("Created container with ID: %s\n", newResp.ID)

	checkpointDir := fmt.Sprintf("/var/lib/docker/containers/%s/checkpoints/checkpoint1", newResp.ID)
	err = os.MkdirAll(checkpointDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("error creating checkpoint directory: %v", err)
	}
	fmt.Println("Created checkpoint directory successfully")

	buf := bytes.NewBuffer(checkpointData)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return "", fmt.Errorf("error creating gzip reader for checkpoint data: %v", err)
	}
	tarReader := tar.NewReader(gz)

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


	err = cli.ContainerStart(context.Background(), newResp.ID, container.StartOptions{CheckpointID: "checkpoint1"})
	if err != nil {
		return "", fmt.Errorf("error starting container: %v", err)
	}
	fmt.Printf("Container started successfully with ID: %s\n", newResp.ID)

	return newResp.ID, nil
}

// currently MigrateContainerToLocalhost is more like to fetch a container from given address to local host
func MigrateContainerToLocalhost(serverAddress string, containerID string) (string, error) {
    conn, err := grpc.Dial(serverAddress, grpc.WithInsecure())
    if err != nil {
        return "", fmt.Errorf("did not connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewContainerMigrationClient(conn)

    startTime := time.Now()

    req := &pb.CheckpointRequest{ContainerId: containerID}
    res, err := client.CheckpointContainer(context.Background(), req)
    if err != nil {
        return "", fmt.Errorf("could not checkpoint container: %v", err)
    }
    fmt.Printf("got checkpoint res")

    volReq := &pb.VolumeRequest{ContainerId: containerID}
	
    volRes, err := client.TransferVolume(context.Background(), volReq)

    if err != nil {
        return "", fmt.Errorf("could not transfer volume: %v", err)
    }
    fmt.Printf("got volume res")
	volumeNameMsg := fmt.Sprintf("the volumename of the container is %s\nthe nfssource of the container is %s\nthe volumedestination of the container is %s", volRes.VolumeName, volRes.NfsSource, volRes.Destination)
	fmt.Print(volumeNameMsg)
	



    volCreateErr := createVolumeFromData(volRes.VolumeName, volRes.VolumeData)
    if volCreateErr != nil {
        return "", fmt.Errorf("could not create volume: %v", volCreateErr)
    }
    fmt.Printf("created volume")


    newContainerID, err := restoreContainer(res.CheckpointData,  volRes.VolumeName)
    if err != nil {
        return "", fmt.Errorf("could not restore container: %v", err)
    }

    endTime := time.Now()
    elapsedTime := endTime.Sub(startTime)
    fmt.Printf("Time taken from checkpointing container to finishing restore: %s\n", elapsedTime)

    return newContainerID, nil
}



