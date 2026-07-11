package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// backupSidecarImage is a minimal image whose bundled tar handles uid/gid,
// permissions and symlinks — the properties a faithful volume copy must keep.
const backupSidecarImage = "busybox:latest"

const stderrTailLimit = 8 * 1024

// StreamContainerMounts streams a gzipped tar of the given container paths.
//
// The bytes come from a throwaway sidecar that inherits the target's mounts via
// VolumesFrom with a :ro suffix, so the source data is never mutated. The
// sidecar runs as root on purpose: a backup must read every uid/gid the target
// wrote, which an unprivileged reader could not.
//
// Closing the returned reader reaps the sidecar. Callers must always Close, even
// on a partial read, or the sidecar leaks.
func (s *Service) StreamContainerMounts(
	ctx context.Context,
	containerID string,
	paths []string,
) (io.ReadCloser, error) {
	if containerID == "" {
		return nil, errors.New("container id is required")
	}

	if len(paths) == 0 {
		return nil, errors.New("at least one mount path is required")
	}

	cli, err := getClient()
	if err != nil {
		return nil, err
	}

	if err := ensureImage(ctx, cli, backupSidecarImage); err != nil {
		return nil, err
	}

	created, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image: backupSidecarImage,
			Cmd:   append([]string{"tar", "-czf", "-"}, paths...),
		},
		HostConfig: &container.HostConfig{
			VolumesFrom: []string{containerID + ":ro"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create backup sidecar: %w", err)
	}

	stream, err := newMountStream(ctx, cli, created.ID)
	if err != nil {
		_ = removeSidecar(ctx, cli, created.ID)

		return nil, err
	}

	return stream, nil
}

func ensureImage(ctx context.Context, cli *client.Client, reference string) error {
	_, err := cli.ImageInspect(ctx, reference)
	if err == nil {
		return nil
	}

	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("inspect image %s: %w", reference, err)
	}

	pull, err := cli.ImagePull(ctx, reference, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", reference, err)
	}

	waitErr := pull.Wait(ctx)
	closeErr := pull.Close()

	if waitErr != nil {
		return fmt.Errorf("pull image %s: %w", reference, waitErr)
	}

	if closeErr != nil {
		return fmt.Errorf("pull image %s: %w", reference, closeErr)
	}

	return nil
}

type mountStream struct {
	cli        *client.Client
	sidecarID  string
	reader     *io.PipeReader
	attach     client.ContainerAttachResult
	stderrTail *tailBuffer
	cancelCopy context.CancelFunc
	closeOnce  sync.Once
	closeErr   error
}

func newMountStream(ctx context.Context, cli *client.Client, sidecarID string) (*mountStream, error) {
	attach, err := cli.ContainerAttach(ctx, sidecarID, client.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("attach backup sidecar: %w", err)
	}

	if _, err := cli.ContainerStart(ctx, sidecarID, client.ContainerStartOptions{}); err != nil {
		attach.Close()

		return nil, fmt.Errorf("start backup sidecar: %w", err)
	}

	reader, writer := io.Pipe()
	stderrTail := &tailBuffer{limit: stderrTailLimit}
	copyCtx, cancelCopy := context.WithCancel(ctx)

	stream := &mountStream{
		cli:        cli,
		sidecarID:  sidecarID,
		reader:     reader,
		attach:     attach,
		stderrTail: stderrTail,
		cancelCopy: cancelCopy,
	}

	go func() {
		context.AfterFunc(copyCtx, attach.Close)

		_, copyErr := stdcopy.StdCopy(writer, stderrTail, attach.Reader)
		writer.CloseWithError(copyErr)
	}()

	return stream, nil
}

func (s *mountStream) Read(buffer []byte) (int, error) {
	return s.reader.Read(buffer)
}

func (s *mountStream) Close() error {
	s.closeOnce.Do(func() {
		_ = s.reader.Close()
		s.cancelCopy()
		s.attach.Close()

		reapErr := s.reapSidecar()
		removeErr := removeSidecar(context.WithoutCancel(context.Background()), s.cli, s.sidecarID)

		s.closeErr = errors.Join(reapErr, removeErr)
	})

	return s.closeErr
}

func (s *mountStream) reapSidecar() error {
	waitCtx := context.WithoutCancel(context.Background())

	waitResult := s.cli.ContainerWait(waitCtx, s.sidecarID, client.ContainerWaitOptions{
		Condition: container.WaitConditionNotRunning,
	})

	select {
	case waitErr := <-waitResult.Error:
		return fmt.Errorf("wait for backup sidecar: %w", waitErr)
	case response := <-waitResult.Result:
		if response.StatusCode != 0 {
			return fmt.Errorf("backup sidecar exited with code %d: %s", response.StatusCode, s.stderrTail.String())
		}

		return nil
	}
}

func removeSidecar(ctx context.Context, cli *client.Client, sidecarID string) error {
	_, err := cli.ContainerRemove(ctx, sidecarID, client.ContainerRemoveOptions{Force: true})
	if err != nil {
		return fmt.Errorf("remove backup sidecar: %w", err)
	}

	return nil
}

type tailBuffer struct {
	limit    int
	buffered []byte
}

func (b *tailBuffer) Write(chunk []byte) (int, error) {
	b.buffered = append(b.buffered, chunk...)
	if len(b.buffered) > b.limit {
		b.buffered = b.buffered[len(b.buffered)-b.limit:]
	}

	return len(chunk), nil
}

func (b *tailBuffer) String() string {
	return strings.TrimSpace(string(b.buffered))
}
