package docker

import "github.com/google/uuid"

type CreateBackupRequestDTO struct {
	ContainerID string          `json:"containerId" binding:"required"`
	StorageID   uuid.UUID       `json:"storageId"`
	MountPaths  []string        `json:"mountPaths"  binding:"required"`
	Consistency ConsistencyMode `json:"consistency"`
	IsEncrypted bool            `json:"isEncrypted"`
}
