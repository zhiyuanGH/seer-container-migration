package Migration

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"context"
)

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