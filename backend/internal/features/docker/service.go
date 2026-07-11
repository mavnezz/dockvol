package docker

import (
	"context"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
)

type Service struct{}

func (s *Service) GetContainers(ctx context.Context) ([]Container, error) {
	summaries, err := listContainers(ctx)
	if err != nil {
		return nil, err
	}

	containers := make([]Container, 0, len(summaries))
	for _, summary := range summaries {
		containers = append(containers, toContainer(summary))
	}

	return containers, nil
}

func (s *Service) findContainer(ctx context.Context, containerID string) (*Container, error) {
	containers, err := s.GetContainers(ctx)
	if err != nil {
		return nil, err
	}

	for i := range containers {
		if containers[i].ID == containerID {
			return &containers[i], nil
		}
	}

	return nil, ErrContainerNotFound
}

func (s *Service) findContainerByName(ctx context.Context, containerName string) (*Container, error) {
	containers, err := s.GetContainers(ctx)
	if err != nil {
		return nil, err
	}

	for i := range containers {
		if containers[i].Name == containerName {
			return &containers[i], nil
		}
	}

	return nil, ErrContainerNotFound
}

func toContainer(summary container.Summary) Container {
	mounts := make([]ContainerMount, 0, len(summary.Mounts))
	for _, mountPoint := range summary.Mounts {
		mounts = append(mounts, toMount(mountPoint))
	}

	return Container{
		ID:     summary.ID,
		Name:   containerName(summary.Names),
		Image:  summary.Image,
		State:  string(summary.State),
		Mounts: mounts,
	}
}

func toMount(mountPoint container.MountPoint) ContainerMount {
	return ContainerMount{
		Type:              string(mountPoint.Type),
		Name:              mountPoint.Name,
		Source:            mountPoint.Source,
		Destination:       mountPoint.Destination,
		IsBackupCandidate: isBackupCandidate(mountPoint),
	}
}

// noiseMountDestinations are the container paths Docker injects that never hold
// user data worth backing up.
var noiseMountDestinations = map[string]bool{
	"/etc/hostname":    true,
	"/etc/hosts":       true,
	"/etc/resolv.conf": true,
	"/etc/localtime":   true,
}

// noiseMountSources are host paths that are infrastructure, not data — most
// importantly the Docker socket a container may mount to talk to the daemon.
var noiseMountSources = map[string]bool{
	"/var/run/docker.sock": true,
	"/run/docker.sock":     true,
}

func isBackupCandidate(mountPoint container.MountPoint) bool {
	if mountPoint.Type != mount.TypeBind && mountPoint.Type != mount.TypeVolume {
		return false
	}

	if noiseMountSources[mountPoint.Source] || noiseMountDestinations[mountPoint.Destination] {
		return false
	}

	return true
}

func containerName(names []string) string {
	if len(names) == 0 {
		return ""
	}

	return strings.TrimPrefix(names[0], "/")
}
