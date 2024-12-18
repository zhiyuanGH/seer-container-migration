package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"


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
func PullImageIfNotExists(cli *client.Client, imageName string) (string, error) {
	ctx := context.Background()

	// Inspect the image to check if it exists locally
	_, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		// Image exists locally
		return "Image already exists locally. No network traffic generated.", nil
	}

	// Image not found locally; proceed to pull
	fmt.Printf("Image %s not found locally. Pulling...\n", imageName)

	// Pull the image
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("could not pull image: %v", err)
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
			return "", fmt.Errorf("error decoding image pull progress: %v", err)
		}

		// Accumulate bytes from progressDetail.current
		totalBytes += progress.ProgressDetail.Current

		// Optionally, print the progress
		fmt.Printf("%v\n", progress)
	}

	// Create a summary of the network traffic
	networkTraffic := fmt.Sprintf(
		"Total bytes pulled: %d bytes",
		totalBytes,
	)

	return networkTraffic, nil
}

func main() {
	// Initialize the Docker client with environment variables and API version negotiation
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error initializing Docker client: %v", err)
	}

	// Specify the Docker image to pull
	imageName := "192.168.116.150:5000/node:esgz" // Replace with your desired image

	// Call the PullImageIfNotExists function
	traffic, err := PullImageIfNotExists(cli, imageName)
	if err != nil {
		log.Fatalf("Error pulling image: %v", err)
	}

	// Print the captured network traffic information
	fmt.Println("Network Traffic During Image Pull:")
	fmt.Println(traffic)
}
