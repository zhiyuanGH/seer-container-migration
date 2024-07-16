package ctrtools

import (
    "context"
    "fmt"
    "io/ioutil"
    "os"
    "os/exec"

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
    if err := ioutil.WriteFile(checkpointDir+"/checkpoint.tar", checkpointData, os.ModePerm); err != nil {
        return "", err
    }

    cmd := exec.Command("tar", "-xvf", checkpointDir+"/checkpoint.tar", "-C", checkpointDir)
    if err := cmd.Run(); err != nil {
        return "", err
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
