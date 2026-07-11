package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"dockvol-backend/internal/features/storages"
	"dockvol-backend/internal/util/encryption"
)

type mountBackupSink interface {
	SaveFile(
		ctx context.Context,
		encryptor encryption.FieldEncryptor,
		logger *slog.Logger,
		fileName string,
		file io.Reader,
	) error
}

// decryptingReadCloser yields decrypted bytes from an underlying encrypted
// storage reader while closing that underlying reader, which the decrypting
// wrapper itself does not own.
type decryptingReadCloser struct {
	io.Reader
	closer io.Closer
}

func (d decryptingReadCloser) Close() error { return d.closer.Close() }

type Backuper struct {
	dockerService    *Service
	storageService   *storages.StorageService
	backupRepository *VolumeBackupRepository
	fieldEncryptor   encryption.FieldEncryptor
	streamCipher     *encryption.StreamCipher
	logger           *slog.Logger
}

func (b *Backuper) CreateBackup(ctx context.Context, request CreateBackupRequestDTO) (*VolumeBackup, error) {
	targetStorage, err := b.storageService.GetStorageByID(request.StorageID)
	if err != nil {
		return nil, ErrStorageNotFound
	}

	targetContainer, err := b.dockerService.findContainer(ctx, request.ContainerID)
	if err != nil {
		return nil, err
	}

	if err := validateMountPaths(targetContainer, request.MountPaths); err != nil {
		return nil, err
	}

	return b.startBackup(backupSpec{
		container:     targetContainer,
		mountPaths:    request.MountPaths,
		targetStorage: targetStorage,
		storageID:     request.StorageID,
		consistency:   request.Consistency,
		isEncrypted:   request.IsEncrypted,
	})
}

// CreateBackupForConfig resolves the container by name because its id changes
// every time it is recreated (e.g. docker compose up).
func (b *Backuper) CreateBackupForConfig(ctx context.Context, config *VolumeBackupConfig) (*VolumeBackup, error) {
	targetStorage, err := b.storageService.GetStorageByID(config.StorageID)
	if err != nil {
		return nil, ErrStorageNotFound
	}

	targetContainer, err := b.dockerService.findContainerByName(ctx, config.ContainerName)
	if err != nil {
		return nil, err
	}

	return b.startBackup(backupSpec{
		container:     targetContainer,
		mountPaths:    config.MountPaths,
		targetStorage: targetStorage,
		storageID:     config.StorageID,
		retentionDays: config.RetentionDays,
		consistency:   config.Consistency,
		isEncrypted:   config.IsEncrypted,
	})
}

func (b *Backuper) ListBackups(containerID string) ([]VolumeBackup, error) {
	return b.backupRepository.List(containerID)
}

// OpenBackup's caller must close the returned reader.
func (b *Backuper) OpenBackup(id uuid.UUID) (*VolumeBackup, io.ReadCloser, error) {
	volumeBackup, err := b.backupRepository.FindByID(id)
	if err != nil {
		return nil, nil, err
	}

	targetStorage, err := b.storageService.GetStorageByID(volumeBackup.StorageID)
	if err != nil {
		return nil, nil, ErrStorageNotFound
	}

	reader, err := targetStorage.GetFile(b.fieldEncryptor, volumeBackup.FileName)
	if err != nil {
		return nil, nil, err
	}

	if !volumeBackup.IsEncrypted {
		return volumeBackup, reader, nil
	}

	decryptingReader, err := b.streamCipher.DecryptingReader(reader)
	if err != nil {
		_ = reader.Close()

		return nil, nil, err
	}

	return volumeBackup, decryptingReadCloser{decryptingReader, reader}, nil
}

func (b *Backuper) RestoreBackup(ctx context.Context, id uuid.UUID) error {
	volumeBackup, reader, err := b.OpenBackup(id)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	logger := b.logger.With("backup_id", id, "container_id", volumeBackup.ContainerID)
	logger.Info("restoring volume backup")

	if err := b.dockerService.RestoreContainerMounts(ctx, volumeBackup.ContainerID, reader); err != nil {
		logger.Error("volume restore failed", "error", err)

		return err
	}

	logger.Info("volume restore completed")

	return nil
}

func (b *Backuper) DeleteBackup(id uuid.UUID) error {
	volumeBackup, err := b.backupRepository.FindByID(id)
	if err != nil {
		return err
	}

	targetStorage, err := b.storageService.GetStorageByID(volumeBackup.StorageID)
	if err == nil {
		if deleteErr := targetStorage.DeleteFile(b.fieldEncryptor, volumeBackup.FileName); deleteErr != nil {
			b.logger.Warn("failed to delete backup file from storage", "backup_id", id, "error", deleteErr)
		}
	}

	return b.backupRepository.Delete(volumeBackup)
}

type backupSpec struct {
	container     *Container
	mountPaths    []string
	targetStorage *storages.Storage
	storageID     uuid.UUID
	retentionDays int
	consistency   ConsistencyMode
	isEncrypted   bool
}

func (b *Backuper) startBackup(spec backupSpec) (*VolumeBackup, error) {
	volumeBackup := &VolumeBackup{
		ID:            uuid.New(),
		ContainerID:   spec.container.ID,
		ContainerName: spec.container.Name,
		Image:         spec.container.Image,
		MountPaths:    spec.mountPaths,
		StorageID:     spec.storageID,
		IsEncrypted:   spec.isEncrypted,
		Status:        BackupStatusRunning,
		CreatedAt:     time.Now().UTC(),
	}
	volumeBackup.GenerateFilename(spec.targetStorage.Name)

	if err := b.backupRepository.Save(volumeBackup); err != nil {
		return nil, fmt.Errorf("save volume backup: %w", err)
	}

	go b.runBackup(volumeBackup, spec)

	return volumeBackup, nil
}

// runBackup runs in its own goroutine so the caller returns before the
// (potentially long) stream finishes; a non-zero retentionDays prunes older
// backups of the same container.
func (b *Backuper) runBackup(volumeBackup *VolumeBackup, spec backupSpec) {
	logger := b.logger.With("backup_id", volumeBackup.ID, "container_id", volumeBackup.ContainerID)

	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("volume backup panicked", "error", recovered)
		}
	}()

	restore, err := b.dockerService.prepareForBackup(
		context.Background(),
		logger,
		volumeBackup.ContainerID,
		spec.consistency,
	)
	if err != nil {
		b.finalizeFailed(logger, volumeBackup, spec.targetStorage, fmt.Errorf("prepare container for backup: %w", err))

		return
	}
	defer restore()

	start := time.Now().UTC()

	streamErr := b.streamToStorage(context.Background(), logger, spec.targetStorage, volumeBackup)

	volumeBackup.BackupDurationMs = time.Since(start).Milliseconds()

	if streamErr != nil {
		b.finalizeFailed(logger, volumeBackup, spec.targetStorage, streamErr)

		return
	}

	volumeBackup.Status = BackupStatusCompleted

	if err := b.backupRepository.Save(volumeBackup); err != nil {
		logger.Error("failed to save completed volume backup", "error", err)

		return
	}

	logger.Info(fmt.Sprintf(
		"volume backup completed: %.2f MB in %d ms",
		volumeBackup.BackupSizeMb,
		volumeBackup.BackupDurationMs,
	))

	if spec.retentionDays > 0 {
		b.applyRetention(logger, volumeBackup.ContainerName, spec.retentionDays)
	}
}

func (b *Backuper) finalizeFailed(
	logger *slog.Logger,
	volumeBackup *VolumeBackup,
	targetStorage *storages.Storage,
	cause error,
) {
	failMessage := cause.Error()
	volumeBackup.Status = BackupStatusFailed
	volumeBackup.FailMessage = &failMessage

	if saveErr := b.backupRepository.Save(volumeBackup); saveErr != nil {
		logger.Error("failed to persist failed volume backup", "error", saveErr)
	}

	if deleteErr := targetStorage.DeleteFile(b.fieldEncryptor, volumeBackup.FileName); deleteErr != nil {
		logger.Warn("failed to delete partial volume backup", "error", deleteErr)
	}

	logger.Error("volume backup failed", "error", cause)
}

func (b *Backuper) applyRetention(logger *slog.Logger, containerName string, retentionDays int) {
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)

	oldBackups, err := b.backupRepository.ListOlderThan(containerName, cutoff)
	if err != nil {
		logger.Warn("retention: failed to list old backups", "error", err)

		return
	}

	for i := range oldBackups {
		if err := b.DeleteBackup(oldBackups[i].ID); err != nil {
			logger.Warn("retention: failed to delete old backup", "backup_id", oldBackups[i].ID, "error", err)
		}
	}
}

func (b *Backuper) streamToStorage(
	ctx context.Context,
	logger *slog.Logger,
	sink mountBackupSink,
	volumeBackup *VolumeBackup,
) error {
	stream, err := b.dockerService.StreamContainerMounts(ctx, volumeBackup.ContainerID, volumeBackup.MountPaths)
	if err != nil {
		return err
	}

	source := io.Reader(stream)
	if volumeBackup.IsEncrypted {
		encryptingReader, encryptErr := b.streamCipher.EncryptingReader(stream)
		if encryptErr != nil {
			_ = stream.Close()

			return encryptErr
		}

		source = encryptingReader
	}

	counter := &byteCounter{source: source}

	saveErr := sink.SaveFile(ctx, b.fieldEncryptor, logger, volumeBackup.FileName, counter)
	closeErr := stream.Close()

	volumeBackup.BackupSizeMb = float64(counter.readBytes) / (1024 * 1024)

	return errors.Join(saveErr, closeErr)
}

func validateMountPaths(targetContainer *Container, requestedPaths []string) error {
	mountedDestinations := make([]string, 0, len(targetContainer.Mounts))
	for _, containerMount := range targetContainer.Mounts {
		mountedDestinations = append(mountedDestinations, containerMount.Destination)
	}

	for _, requestedPath := range requestedPaths {
		if !slices.Contains(mountedDestinations, requestedPath) {
			return fmt.Errorf("%w: %s", ErrMountPathNotFound, requestedPath)
		}
	}

	return nil
}
