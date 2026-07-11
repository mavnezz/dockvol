package docker

import (
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/stretchr/testify/assert"
)

func Test_isBackupCandidate_KeepsDataMounts_ExcludesInfrastructure(t *testing.T) {
	cases := []struct {
		name              string
		mountPoint        container.MountPoint
		isBackupCandidate bool
	}{
		{
			name: "named volume",
			mountPoint: container.MountPoint{
				Type:        mount.TypeVolume,
				Source:      "/var/lib/docker/volumes/data/_data",
				Destination: "/data",
			},
			isBackupCandidate: true,
		},
		{
			name: "bind mount with real data",
			mountPoint: container.MountPoint{
				Type:        mount.TypeBind,
				Source:      "/home/user/app",
				Destination: "/app",
			},
			isBackupCandidate: true,
		},
		{
			name: "docker socket",
			mountPoint: container.MountPoint{
				Type:        mount.TypeBind,
				Source:      "/var/run/docker.sock",
				Destination: "/var/run/docker.sock",
			},
			isBackupCandidate: false,
		},
		{
			name: "injected resolv.conf",
			mountPoint: container.MountPoint{
				Type:        mount.TypeBind,
				Source:      "/somewhere",
				Destination: "/etc/resolv.conf",
			},
			isBackupCandidate: false,
		},
		{
			name: "tmpfs",
			mountPoint: container.MountPoint{
				Type:        mount.TypeTmpfs,
				Destination: "/tmp",
			},
			isBackupCandidate: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.isBackupCandidate, isBackupCandidate(testCase.mountPoint))
		})
	}
}

func Test_containerName_StripsLeadingSlash(t *testing.T) {
	assert.Equal(t, "minio", containerName([]string{"/minio"}))
	assert.Equal(t, "", containerName(nil))
}
