package ctrtools

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

    "google.golang.org/grpc"
    pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/client"
)

func restoreContainer(checkpointData []byte) (string, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return "", err
    }

    newResp, err := cli.ContainerCreate(context.Background(), &container.Config{
        Image: "busybox",
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

func MigrateContainer(serverAddress string, containerID string) (string, error) {
    conn, err := grpc.Dial(serverAddress, grpc.WithInsecure())
    if err != nil {
        return "", fmt.Errorf("did not connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewContainerMigrationClient(conn)

    req := &pb.CheckpointRequest{ContainerId: containerID}
    res, err := client.CheckpointContainer(context.Background(), req)
    if err != nil {
        return "", fmt.Errorf("could not checkpoint container: %v", err)
    }

    newContainerID, err := restoreContainer(res.CheckpointData)
    if err != nil {
        return "", fmt.Errorf("could not restore container: %v", err)
    }

    return newContainerID, nil
}
