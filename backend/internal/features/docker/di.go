package docker

import (
	"dockvol-backend/internal/features/storages"
	"dockvol-backend/internal/util/encryption"
	"dockvol-backend/internal/util/logger"
)

var dockerService = &Service{}

var volumeBackupRepository = &VolumeBackupRepository{}

var volumeBackupConfigRepository = &VolumeBackupConfigRepository{}

var backuper = &Backuper{
	dockerService,
	storages.GetStorageService(),
	volumeBackupRepository,
	encryption.GetFieldEncryptor(),
	encryption.GetStreamCipher(),
	logger.GetLogger(),
}

var configService = &ConfigService{
	volumeBackupConfigRepository,
}

var backupScheduler = &Scheduler{
	configRepository: volumeBackupConfigRepository,
	backuper:         backuper,
	logger:           logger.GetLogger(),
}

var dockerController = &Controller{
	dockerService,
	backuper,
	configService,
}

func GetDockerService() *Service { return dockerService }

func GetDockerController() *Controller { return dockerController }

func GetBackupScheduler() *Scheduler { return backupScheduler }
