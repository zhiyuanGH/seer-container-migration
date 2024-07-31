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

    "google.golang.org/grpc"
    pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/image"
    "github.com/docker/docker/api/types/filters"
    "github.com/docker/docker/client"
    "os/exec"
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

func restoreContainer(checkpointData []byte) (string, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return "", err
    }

    imageName := "busybox"
    err = PullImageIfNotExists(cli, imageName)
    if err != nil {
        return "", err
    }

    newResp, err := cli.ContainerCreate(context.Background(), &container.Config{
        Image: imageName,
        Cmd:   []string{"sh", "-c", "i=0; while true; do echo $i; i=$((i+1)); sleep 1; done"},
        Tty:   false,
    }, nil, nil, nil, "")
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

    if err := cli.ContainerStart(context.Background(), newResp.ID, container.StartOptions{CheckpointID: "checkpoint1"}); err != nil {
        return "", err
    }

    return newResp.ID, nil
}



//currently MigrateContainer is more like to fetch a container from given address to local host 
func MigrateContainer(serverAddress string, containerID string) (string, error) {
    conn, err := grpc.Dial(serverAddress, grpc.WithInsecure())
    if (err != nil) {
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

    newContainerID, err := restoreContainer(res.CheckpointData)
    if err != nil {
        return "", fmt.Errorf("could not restore container: %v", err)
    }

    endTime := time.Now()
    elapsedTime := endTime.Sub(startTime)
    fmt.Printf("Time taken from checkpointing container to finishing restore: %s\n", elapsedTime)

    return newContainerID, nil
}

// ResetSnapshotters resets the stargz and overlayfs snapshotters by performing the following steps:
// 1. Stops all containers.
// 2. Removes all containers.
// 3. Prunes all images.
// 4. Removes all images.
// 5. Restarts the stargz-snapshotter service.
// 6. Stops the stargz-snapshotter service.
// 7. Clears stargz snapshotter data.
// 8. Restarts the stargz-snapshotter service again.
// 9. Clears overlayfs snapshotter data.
// 10. Restarts the containerd service.
// This function returns an error if any of the steps fail.

func ResetSnapshotters() error {
    fmt.Println("Resetting stargz and overlayfs snapshotters...")

    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return err
    }

    // Stop all containers
    fmt.Println("Stopping all containers...")
    containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
    if err != nil {
        return err
    }

    for _, per_container := range containers {
        if err := cli.ContainerStop(context.Background(), per_container.ID, container.StopOptions{}); err != nil {
            return err
        }
    }
    time.Sleep(1 * time.Second)

    fmt.Println("Removing all containers...")
    for _, per_container := range containers {
        if err := cli.ContainerRemove(context.Background(), per_container.ID, container.RemoveOptions{Force: true}); err != nil {
            return err
        }
    }
    time.Sleep(1 * time.Second)

    imagesBeforePrune, err := cli.ImageList(context.Background(), image.ListOptions{All: true})
    if err != nil {
        return err
    }
    fmt.Println("Images before pruning:")
    for _, image := range imagesBeforePrune {
        fmt.Printf("Image: %s\n", image.ID)
    }

    fmt.Println("Pruning all images...")
    pruneReport, err := cli.ImagesPrune(context.Background(), filters.Args{})
    if err != nil {
        return err
    }
    fmt.Printf("Pruned images: %v\n", pruneReport.ImagesDeleted)

    imagesAfterPrune, err := cli.ImageList(context.Background(), image.ListOptions{All: true})
    if err != nil {
        return err
    }
    for _, per_image := range imagesAfterPrune {
        fmt.Printf("Force removing image %s\n", per_image.ID)
        if _, err := cli.ImageRemove(context.Background(), per_image.ID, image.RemoveOptions{Force: true}); err != nil {
            return err
        }
    }

    // Reset stargz snapshotter
    fmt.Println("Restarting stargz-snapshotter service...")
    if err := runCommand("systemctl restart stargz-snapshotter"); err != nil {
        return err
    }
    time.Sleep(1 * time.Second)

    fmt.Println("Stopping stargz-snapshotter service...")
    if err := runCommand("systemctl stop stargz-snapshotter"); err != nil {
        return err
    }
    time.Sleep(1 * time.Second)

    fmt.Println("Clearing stargz snapshotter data...")
    if err := runCommand("rm -rf /var/lib/containerd-stargz-grpc/*"); err != nil {
        return err
    }
    time.Sleep(1 * time.Second)

    fmt.Println("Restarting stargz-snapshotter service again...")
    if err := runCommand("systemctl restart stargz-snapshotter"); err != nil {
        return err
    }
    time.Sleep(1 * time.Second)

    // Reset overlayfs snapshotter
    fmt.Println("Clearing overlayfs snapshotter data...")
    if err := runCommand("rm -rf /var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/*"); err != nil {
        return err
    }
    time.Sleep(1 * time.Second)

    fmt.Println("Restarting containerd service...")
    if err := runCommand("systemctl restart containerd"); err != nil {
        return err
    }
    time.Sleep(1 * time.Second)

    fmt.Println("Reset complete.")
    return nil
}

func runCommand(command string) error {
    fullCommand := fmt.Sprintf("echo 'gh' | sudo -S -k bash -c \"%s\"", command)
    cmd := exec.Command("bash", "-c", fullCommand)
    output, err := cmd.CombinedOutput()
    fmt.Printf("Running command: %s\nOutput: %s\n", command, output)
    if err != nil {
        return fmt.Errorf("command failed: %s, error: %w", command, err)
    }
    return nil
}
