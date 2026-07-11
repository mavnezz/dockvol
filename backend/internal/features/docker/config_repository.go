package docker

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"dockvol-backend/internal/storage"
)

type VolumeBackupConfigRepository struct{}

func (r *VolumeBackupConfigRepository) Save(config *VolumeBackupConfig) error {
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}

	return storage.GetDb().Save(config).Error
}

func (r *VolumeBackupConfigRepository) FindByID(id uuid.UUID) (*VolumeBackupConfig, error) {
	var config VolumeBackupConfig

	result := storage.GetDb().Where("id = ?", id).First(&config)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("backup config not found")
		}

		return nil, result.Error
	}

	return &config, nil
}

func (r *VolumeBackupConfigRepository) FindByContainerName(containerName string) (*VolumeBackupConfig, error) {
	var config VolumeBackupConfig

	result := storage.GetDb().Where("container_name = ?", containerName).First(&config)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &config, nil
}

func (r *VolumeBackupConfigRepository) FindAll() ([]VolumeBackupConfig, error) {
	var configs []VolumeBackupConfig
	if err := storage.GetDb().Order("container_name ASC").Find(&configs).Error; err != nil {
		return nil, err
	}

	return configs, nil
}

func (r *VolumeBackupConfigRepository) FindDue(now time.Time) ([]VolumeBackupConfig, error) {
	var configs []VolumeBackupConfig

	err := storage.GetDb().
		Where("is_enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", true, now).
		Find(&configs).Error
	if err != nil {
		return nil, err
	}

	return configs, nil
}

func (r *VolumeBackupConfigRepository) Delete(config *VolumeBackupConfig) error {
	return storage.GetDb().Delete(config).Error
}
