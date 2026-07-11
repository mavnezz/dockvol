package docker

import (
	"fmt"
	"path"
	"time"

	"github.com/google/uuid"

	files_utils "dockvol-backend/internal/util/files"
)

type VolumeBackup struct {
	ID       uuid.UUID `json:"id"       gorm:"column:id;primaryKey"`
	FileName string    `json:"fileName" gorm:"column:file_name;type:text;not null"`

	ContainerID   string `json:"containerId"   gorm:"column:container_id;type:text;not null"`
	ContainerName string `json:"containerName" gorm:"column:container_name;type:text;not null"`
	Image         string `json:"image"         gorm:"column:image;type:text;not null"`

	MountPaths []string `json:"mountPaths" gorm:"column:mount_paths;serializer:json;not null"`

	StorageID uuid.UUID `json:"storageId" gorm:"column:storage_id;not null"`

	Status      BackupStatus `json:"status"      gorm:"column:status;type:text;not null"`
	FailMessage *string      `json:"failMessage" gorm:"column:fail_message;type:text"`

	IsEncrypted bool `json:"isEncrypted" gorm:"column:is_encrypted;default:false"`

	BackupSizeMb     float64 `json:"backupSizeMb"     gorm:"column:backup_size_mb;default:0"`
	BackupDurationMs int64   `json:"backupDurationMs" gorm:"column:backup_duration_ms;default:0"`

	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
}

func (VolumeBackup) TableName() string { return "volume_backups" }

// GenerateFilename groups a backup under a folder named after its target storage
// so a storage medium shared by several storages keeps each one's backups apart.
func (b *VolumeBackup) GenerateFilename(storageName string) {
	timestamp := time.Now().UTC()

	fileName := fmt.Sprintf(
		"%s-%s-%s.tar.gz",
		files_utils.SanitizeFilename(b.ContainerName),
		timestamp.Format("20060102-150405"),
		b.ID.String(),
	)

	b.FileName = path.Join(files_utils.SanitizeFilename(storageName), fileName)
}
