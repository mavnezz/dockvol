package docker

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// RestoreContainerMounts mounts the target's volumes read-write (unlike a
// backup) so the extracted archive overwrites the live data, and the sidecar
// runs as root so it can write back every uid/gid the archive carries.
func (s *Service) RestoreContainerMounts(ctx context.Context, containerID string, archive io.Reader) error {
	if containerID == "" {
		return errors.New("container id is required")
	}

	cli, err := getClient()
	if err != nil {
		return err
	}

	if err := ensureImage(ctx, cli, backupSidecarImage); err != nil {
		return err
	}

	created, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image:       backupSidecarImage,
			Cmd:         []string{"tar", "-xzf", "-", "-C", "/"},
			OpenStdin:   true,
			StdinOnce:   true,
			AttachStdin: true,
		},
		HostConfig: &container.HostConfig{
			VolumesFrom: []string{containerID},
		},
	})
	if err != nil {
		return fmt.Errorf("create restore sidecar: %w", err)
	}
	defer func() {
		_ = removeSidecar(context.WithoutCancel(context.Background()), cli, created.ID)
	}()

	attach, err := cli.ContainerAttach(ctx, created.ID, client.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return fmt.Errorf("attach restore sidecar: %w", err)
	}
	defer attach.Close()

	if _, err := cli.ContainerStart(ctx, created.ID, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("start restore sidecar: %w", err)
	}

	stderrTail := &tailBuffer{limit: stderrTailLimit}
	drained := make(chan struct{})
	go func() {
		_, _ = stdcopy.StdCopy(io.Discard, stderrTail, attach.Reader)
		close(drained)
	}()

	if _, err := io.Copy(attach.Conn, archive); err != nil {
		return fmt.Errorf("write archive to restore sidecar: %w", err)
	}

	if err := attach.CloseWrite(); err != nil {
		return fmt.Errorf("close restore sidecar stdin: %w", err)
	}

	waitResult := cli.ContainerWait(ctx, created.ID, client.ContainerWaitOptions{
		Condition: container.WaitConditionNotRunning,
	})

	select {
	case waitErr := <-waitResult.Error:
		return fmt.Errorf("wait for restore sidecar: %w", waitErr)
	case response := <-waitResult.Result:
		<-drained
		if response.StatusCode != 0 {
			return fmt.Errorf("restore sidecar exited with code %d: %s", response.StatusCode, stderrTail.String())
		}

		return nil
	}
}
