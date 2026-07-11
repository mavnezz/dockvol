package docker

import (
	"context"
	"sync"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// The client connects lazily, so a missing daemon surfaces as an error on the
// first call rather than at construction.
var (
	dockerClient     *client.Client
	dockerClientOnce sync.Once
	errDockerClient  error
)

func getClient() (*client.Client, error) {
	dockerClientOnce.Do(func() {
		dockerClient, errDockerClient = client.New(client.FromEnv)
	})

	return dockerClient, errDockerClient
}

func listContainers(ctx context.Context) ([]container.Summary, error) {
	cli, err := getClient()
	if err != nil {
		return nil, err
	}

	result, err := cli.ContainerList(ctx, client.ContainerListOptions{All: false})
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}
