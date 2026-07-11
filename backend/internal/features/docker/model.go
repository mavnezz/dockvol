package docker

// A backup selects a container and reads its mounts, rather than tracking
// individual volumes.
type Container struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Image  string           `json:"image"`
	State  string           `json:"state"`
	Mounts []ContainerMount `json:"mounts"`
}

type ContainerMount struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Destination string `json:"destination"`

	// Pre-selects real data mounts and excludes infrastructure mounts (the
	// Docker socket, injected system files) that hold no user data.
	IsBackupCandidate bool `json:"isBackupCandidate"`
}
