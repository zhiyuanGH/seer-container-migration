package Migration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"



	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// ImagePullProgress represents the structure of progress messages from Docker ImagePull.
type ImagePullProgress struct {
	Status         string `json:"status"`
	ID             string `json:"id,omitempty"`
	Progress       string `json:"progress,omitempty"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail,omitempty"`
	Digest string `json:"digest,omitempty"`
	Error  string `json:"error,omitempty"`
}

// PullImageIfNotExists checks if a Docker image exists locally.
// If not, it pulls the image and returns the total bytes pulled.
func PullImageIfNotExists(cli *client.Client, imageName string) (int64, error) {
	ctx := context.Background()

	// Inspect the image to check if it exists locally
	_, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		// Image exists locally
		return 0, nil
	}

	// Image not found locally; proceed to pull
	fmt.Printf("Image %s not found locally. Pulling...\n", imageName)

	// Pull the image
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return 0, fmt.Errorf("could not pull image: %v", err)
	}
	defer reader.Close()

	// Initialize total bytes pulled
	var totalBytes int64 = 0

	// Create a JSON decoder
	decoder := json.NewDecoder(reader)

	// Read the response line by line
	for {
		var progress ImagePullProgress
		if err := decoder.Decode(&progress); err == io.EOF {
			break
		} else if err != nil {
			return 0, fmt.Errorf("error decoding image pull progress: %v", err)
		}

		// Accumulate bytes from progressDetail.current
		totalBytes += progress.ProgressDetail.Current

		// Optionally, print the progress
		fmt.Printf("%v\n", progress)
	}

	// Create a summary of the network traffic


	return totalBytes, nil
}