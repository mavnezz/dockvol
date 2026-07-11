package docker

import (
	"testing"
)

func Test_GetContainers_ListsHostContainersWithMounts(t *testing.T) {
	containers, err := (&Service{}).GetContainers(t.Context())
	if err != nil {
		t.Skipf("docker daemon not reachable: %v", err)
	}

	t.Logf("discovered %d containers", len(containers))
	for _, discoveredContainer := range containers {
		t.Logf(
			"  %s image=%s state=%s mounts=%d",
			discoveredContainer.Name,
			discoveredContainer.Image,
			discoveredContainer.State,
			len(discoveredContainer.Mounts),
		)

		for _, containerMount := range discoveredContainer.Mounts {
			t.Logf(
				"    [%s] %s -> %s candidate=%v",
				containerMount.Type,
				containerMount.Source,
				containerMount.Destination,
				containerMount.IsBackupCandidate,
			)
		}
	}
}
