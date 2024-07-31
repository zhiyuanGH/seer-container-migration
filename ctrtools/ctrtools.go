package ctrtools

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
    "github.com/docker/docker/api/types/volume"
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

// func restoreContainer(checkpointData []byte) (string, error) {
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
// 	if err != nil {
// 		return "", err
// 	}

// 	imageName := "ghcr.io/stargz-containers/golang:1.18-esgz"
// 	err = PullImageIfNotExists(cli, imageName)
// 	if err != nil {
// 		return "", err
// 	}

// 	newResp, err := cli.ContainerCreate(context.Background(), &container.Config{
// 		Image: imageName,
// 		Cmd:   []string{"sh", "-c", "i=0; while true; do echo $i; i=$((i+1)); sleep 1; done"},
// 		Tty:   false,
// 	}, nil, nil, nil, "")
// 	if err != nil {
// 		return "", err
// 	}

// 	checkpointDir := fmt.Sprintf("/var/lib/docker/containers/%s/checkpoints/checkpoint1", newResp.ID)
// 	os.MkdirAll(checkpointDir, os.ModePerm)

// 	buf := bytes.NewBuffer(checkpointData)
// 	gz, err := gzip.NewReader(buf)
// 	if err != nil {
// 		return "", err
// 	}
// 	tarReader := tar.NewReader(gz)

// 	for {
// 		hdr, err := tarReader.Next()
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			return "", err
// 		}

// 		target := filepath.Join(checkpointDir, hdr.Name)
// 		if hdr.Typeflag == tar.TypeDir {
// 			if err := os.MkdirAll(target, os.ModePerm); err != nil {
// 				return "", err
// 			}
// 		} else {
// 			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
// 			if err != nil {
// 				return "", err
// 			}
// 			if _, err := io.Copy(f, tarReader); err != nil {
// 				return "", err
// 			}
// 			f.Close()
// 		}
// 	}

// 	if err := cli.ContainerStart(context.Background(), newResp.ID, container.StartOptions{CheckpointID: "checkpoint1"}); err != nil {
// 		return "", err
// 	}

// 	return newResp.ID, nil
// }

func restoreContainer(checkpointData []byte, volumeData []byte, volumeName string) (string, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return "", err
    }

    imageName := "ghcr.io/stargz-containers/golang:1.18-esgz"
    err = PullImageIfNotExists(cli, imageName)
    if err != nil {
        return "", err
    }

    newResp, err := cli.ContainerCreate(context.Background(), &container.Config{
        Image: imageName,
        Cmd:   []string{"sh", "-c", "i=0; while true; do echo $i; i=$((i+1)); sleep 1; done"},
        Tty:   false,
    }, &container.HostConfig{
        Binds: []string{fmt.Sprintf("%s:/mnt/%s", volumeName, volumeName)},
    }, nil, nil, "")
    if err != nil {
        return "", err
    }

    checkpointDir := fmt.Sprintf("/var/lib/docker/containers/%s/checkpoints/checkpoint1", newResp.ID)
    os.MkdirAll(checkpointDir, os.ModePerm)

    buf := bytes.NewBuffer(checkpointData)
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

        target := filepath.Join(checkpointDir, hdr.Name)
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

    volumeDir := fmt.Sprintf("/mnt/%s", volumeName)
    os.MkdirAll(volumeDir, os.ModePerm)

    buf = bytes.NewBuffer(volumeData)
    gz, err = gzip.NewReader(buf)
    if err != nil {
        return "", err
    }
    tarReader = tar.NewReader(gz)

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

    if err := cli.ContainerStart(context.Background(), newResp.ID, container.StartOptions{CheckpointID: "checkpoint1"}); err != nil {
        return "", err
    }

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

    volReq := &pb.VolumeRequest{ContainerId: containerID}
    volRes, err := client.TransferVolume(context.Background(), volReq)
    if err != nil {
        return "", fmt.Errorf("could not transfer volume: %v", err)
    }

    volCreateErr := createVolumeFromData(volRes.VolumeName, volRes.VolumeData)
    if volCreateErr != nil {
        return "", fmt.Errorf("could not create volume: %v", volCreateErr)
    }



    newContainerID, err := restoreContainer(res.CheckpointData, volRes.VolumeData, volRes.VolumeName)
    if err != nil {
        return "", fmt.Errorf("could not restore container: %v", err)
    }

    endTime := time.Now()
    elapsedTime := endTime.Sub(startTime)
    fmt.Printf("Time taken from checkpointing container to finishing restore: %s\n", elapsedTime)

    return newContainerID, nil
}


func createVolumeFromData(volumeName string, volumeData []byte) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	_, err = cli.VolumeCreate(context.Background(), volume.CreateOptions{
		Name: volumeName,
	})
	if err != nil {
		return err
	}

	volumeDir := fmt.Sprintf("/var/lib/docker/volumes/%s/_data", volumeName)
	os.MkdirAll(volumeDir, os.ModePerm)

	buf := bytes.NewBuffer(volumeData)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(gz)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(volumeDir, hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				return err
			}
		} else {
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tarReader); err != nil {
				return err
			}
			f.Close()
		}
	}

	return nil
}