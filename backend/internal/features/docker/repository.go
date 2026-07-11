package docker

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"dockvol-backend/internal/storage"
)

type VolumeBackupRepository struct{}

func (r *VolumeBackupRepository) Save(volumeBackup *VolumeBackup) error {
	if volumeBackup.ID == uuid.Nil {
		volumeBackup.ID = uuid.New()
	}

	return storage.GetDb().Save(volumeBackup).Error
}

func (r *VolumeBackupRepository) FindByID(id uuid.UUID) (*VolumeBackup, error) {
	var volumeBackup VolumeBackup

	result := storage.GetDb().Where("id = ?", id).First(&volumeBackup)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("volume backup not found")
		}

		return nil, result.Error
	}

	return &volumeBackup, nil
}

func (r *VolumeBackupRepository) List(containerID string) ([]VolumeBackup, error) {
	query := storage.GetDb().Order("created_at DESC")
	if containerID != "" {
		query = query.Where("container_id = ?", containerID)
	}

	var volumeBackups []VolumeBackup
	if err := query.Find(&volumeBackups).Error; err != nil {
		return nil, err
	}

	return volumeBackups, nil
}

func (r *VolumeBackupRepository) ListOlderThan(containerName string, cutoff time.Time) ([]VolumeBackup, error) {
	var volumeBackups []VolumeBackup

	err := storage.GetDb().
		Where("container_name = ? AND created_at < ?", containerName, cutoff).
		Find(&volumeBackups).Error
	if err != nil {
		return nil, err
	}

	return volumeBackups, nil
}

func (r *VolumeBackupRepository) Delete(volumeBackup *VolumeBackup) error {
	return storage.GetDb().Delete(volumeBackup).Error
}
