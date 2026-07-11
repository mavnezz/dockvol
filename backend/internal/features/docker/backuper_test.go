package docker

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"dockvol-backend/internal/util/encryption"
)

// validatingSink stands in for a real storage: it verifies the streamed payload
// is a valid gzipped tar and records the byte count, without persisting or
// buffering the whole archive — so the test needs no storage config or database.
type validatingSink struct {
	receivedFileName string
	fileEntryCount   int
	consumedBytes    int64
}

func (s *validatingSink) SaveFile(
	_ context.Context,
	_ encryption.FieldEncryptor,
	_ *slog.Logger,
	fileName string,
	file io.Reader,
) error {
	s.receivedFileName = fileName

	counter := &byteCounter{source: file}

	gzipReader, err := gzip.NewReader(counter)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeReg {
			s.fileEntryCount++
		}
	}

	if _, err := io.Copy(io.Discard, gzipReader); err != nil {
		return err
	}

	s.consumedBytes = counter.readBytes

	return nil
}

func Test_StreamToStorage_WritesValidGzipTarOfContainerData_ToSink(t *testing.T) {
	dockerService := &Service{}

	containers, err := dockerService.GetContainers(t.Context())
	if err != nil {
		t.Skipf("docker unreachable: %v", err)
	}

	backupTarget, backupPaths := findBackupCandidate(containers, ownContainerID())
	if backupTarget.ID == "" {
		t.Skip("no running container with a backup-candidate mount on this host")
	}

	backuper := &Backuper{
		dockerService: dockerService,
		logger:        slog.New(slog.DiscardHandler),
	}

	volumeBackup := &VolumeBackup{
		ID:          uuid.New(),
		ContainerID: backupTarget.ID,
		MountPaths:  backupPaths,
		FileName:    "test-volume-backup.tar.gz",
	}

	sink := &validatingSink{}

	require.NoError(t, backuper.streamToStorage(t.Context(), slog.New(slog.DiscardHandler), sink, volumeBackup))

	require.Equal(t, volumeBackup.FileName, sink.receivedFileName)
	require.Positive(t, sink.fileEntryCount, "expected at least one file entry in the streamed tar")
	require.Positive(t, volumeBackup.BackupSizeMb, "expected a non-zero backup size")
	require.InDelta(t, float64(sink.consumedBytes)/(1024*1024), volumeBackup.BackupSizeMb, 1e-9)

	t.Logf(
		"streamed %d bytes (%.4f MB) across %d file entries from %q (%s)",
		sink.consumedBytes,
		volumeBackup.BackupSizeMb,
		sink.fileEntryCount,
		backupTarget.Name,
		backupTarget.Image,
	)
}
