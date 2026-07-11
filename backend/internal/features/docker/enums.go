package docker

type BackupStatus string

const (
	BackupStatusRunning   BackupStatus = "RUNNING"
	BackupStatusCompleted BackupStatus = "COMPLETED"
	BackupStatusFailed    BackupStatus = "FAILED"
)

// ConsistencyMode controls whether the container keeps running while its mounts
// are read. Pausing or stopping it yields a consistent snapshot for apps that
// write continuously (e.g. databases).
type ConsistencyMode string

const (
	ConsistencyModeNone  ConsistencyMode = "NONE"
	ConsistencyModePause ConsistencyMode = "PAUSE"
	ConsistencyModeStop  ConsistencyMode = "STOP"
)
