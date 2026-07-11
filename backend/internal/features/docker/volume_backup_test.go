package docker

import (
	"path"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openVolumeBackupTestDb(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "volume_backups.db")
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)"

	testDb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, testDb.AutoMigrate(&VolumeBackup{}))

	return testDb
}

func Test_VolumeBackup_MigratesAndRoundTripsMountPaths_OnSqlite(t *testing.T) {
	testDb := openVolumeBackupTestDb(t)

	persistedBackup := &VolumeBackup{
		ID:            uuid.New(),
		ContainerID:   "abc123def456",
		ContainerName: "minio",
		Image:         "minio/minio",
		MountPaths:    []string{"/data", "/root/.minio"},
		StorageID:     uuid.New(),
		Status:        BackupStatusCompleted,
		BackupSizeMb:  12.5,
	}
	persistedBackup.GenerateFilename("Local Disk")

	require.NoError(t, testDb.Create(persistedBackup).Error)

	var reloadedBackup VolumeBackup
	require.NoError(t, testDb.Where("id = ?", persistedBackup.ID).First(&reloadedBackup).Error)

	assert.Equal(t, persistedBackup.MountPaths, reloadedBackup.MountPaths)
	assert.Equal(t, "minio", reloadedBackup.ContainerName)
	assert.Equal(t, BackupStatusCompleted, reloadedBackup.Status)
	assert.InDelta(t, 12.5, reloadedBackup.BackupSizeMb, 0.0001)
	assert.Contains(t, reloadedBackup.FileName, ".tar.gz")
	assert.Equal(t, "Local_Disk", path.Dir(reloadedBackup.FileName))
}
