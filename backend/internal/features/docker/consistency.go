package docker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/moby/moby/client"
)

// prepareForBackup applies the consistency mode before a backup reads a
// container's mounts and returns a restore function that undoes it. The restore
// runs on a cancellation-detached context so a paused or stopped container is
// always brought back, even if the backup fails; a failure to bring it back is
// logged at Error because it leaves the container down.
func (s *Service) prepareForBackup(
	ctx context.Context,
	logger *slog.Logger,
	containerID string,
	mode ConsistencyMode,
) (func(), error) {
	cli, err := getClient()
	if err != nil {
		return nil, err
	}

	switch mode {
	case ConsistencyModePause:
		if _, err := cli.ContainerPause(ctx, containerID, client.ContainerPauseOptions{}); err != nil {
			return nil, fmt.Errorf("pause container: %w", err)
		}

		return func() {
			if _, err := cli.ContainerUnpause(
				context.WithoutCancel(ctx),
				containerID,
				client.ContainerUnpauseOptions{},
			); err != nil {
				logger.Error("failed to unpause container after backup", "error", err)
			}
		}, nil

	case ConsistencyModeStop:
		if _, err := cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{}); err != nil {
			return nil, fmt.Errorf("stop container: %w", err)
		}

		return func() {
			if _, err := cli.ContainerStart(
				context.WithoutCancel(ctx),
				containerID,
				client.ContainerStartOptions{},
			); err != nil {
				logger.Error("failed to start container after backup", "error", err)
			}
		}, nil

	default:
		return func() {}, nil
	}
}
