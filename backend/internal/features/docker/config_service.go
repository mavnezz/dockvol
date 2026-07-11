package docker

import (
	"time"

	"github.com/google/uuid"
)

type ConfigService struct {
	configRepository *VolumeBackupConfigRepository
}

// SaveConfig upserts a container's schedule (one config per container) and
// (re)computes its next run so the scheduler picks it up.
func (s *ConfigService) SaveConfig(config *VolumeBackupConfig) error {
	if config.ID == uuid.Nil {
		existing, err := s.configRepository.FindByContainerName(config.ContainerName)
		if err != nil {
			return err
		}

		if existing != nil {
			config.ID = existing.ID
			config.CreatedAt = existing.CreatedAt
		}
	}

	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now().UTC()
	}

	nextRun := config.NextRun(time.Now().UTC())
	config.NextRunAt = &nextRun

	return s.configRepository.Save(config)
}

func (s *ConfigService) ListConfigs() ([]VolumeBackupConfig, error) {
	return s.configRepository.FindAll()
}

func (s *ConfigService) DeleteConfig(id uuid.UUID) error {
	config, err := s.configRepository.FindByID(id)
	if err != nil {
		return err
	}

	return s.configRepository.Delete(config)
}
