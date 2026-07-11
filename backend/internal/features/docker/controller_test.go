package docker_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dockvol-backend/internal/features/docker"
	"dockvol-backend/internal/util/testutil"
)

type backupsResponse struct {
	Backups []docker.VolumeBackup `json:"backups"`
}

func Test_GetBackups_ReturnsSeededBackupsNewestFirst(t *testing.T) {
	database := testutil.SetupDb(t)
	router := testutil.NewRouter()
	user := testutil.SignUpTestUser(t, router)

	now := time.Now().UTC()
	require.NoError(t, database.Create(&docker.VolumeBackup{
		ID: uuid.New(), ContainerID: "c", ContainerName: "c", MountPaths: []string{"/data"},
		StorageID: uuid.New(), Status: docker.BackupStatusCompleted, FileName: "old.tar.gz",
		CreatedAt: now.Add(-time.Hour),
	}).Error)
	require.NoError(t, database.Create(&docker.VolumeBackup{
		ID: uuid.New(), ContainerID: "c", ContainerName: "c", MountPaths: []string{"/data"},
		StorageID: uuid.New(), Status: docker.BackupStatusCompleted, FileName: "new.tar.gz",
		CreatedAt: now,
	}).Error)

	var response backupsResponse
	testutil.MakeGetAndUnmarshal(t, router, "/api/v1/docker/backups", user.Token, http.StatusOK, &response)

	require.Len(t, response.Backups, 2)
	assert.Equal(t, "new.tar.gz", response.Backups[0].FileName)
}

func Test_GetBackups_FilteredByContainerID(t *testing.T) {
	database := testutil.SetupDb(t)
	router := testutil.NewRouter()
	user := testutil.SignUpTestUser(t, router)

	now := time.Now().UTC()
	require.NoError(t, database.Create(&docker.VolumeBackup{
		ID: uuid.New(), ContainerID: "container-a", ContainerName: "a", MountPaths: []string{"/data"},
		StorageID: uuid.New(), Status: docker.BackupStatusCompleted, FileName: "a.tar.gz", CreatedAt: now,
	}).Error)
	require.NoError(t, database.Create(&docker.VolumeBackup{
		ID: uuid.New(), ContainerID: "container-b", ContainerName: "b", MountPaths: []string{"/data"},
		StorageID: uuid.New(), Status: docker.BackupStatusCompleted, FileName: "b.tar.gz", CreatedAt: now,
	}).Error)

	var response backupsResponse
	testutil.MakeGetAndUnmarshal(
		t, router,
		"/api/v1/docker/backups?containerId=container-a",
		user.Token, http.StatusOK, &response,
	)

	require.Len(t, response.Backups, 1)
	assert.Equal(t, "container-a", response.Backups[0].ContainerID)
}

func Test_DeleteBackup_RemovesTheRecord(t *testing.T) {
	database := testutil.SetupDb(t)
	router := testutil.NewRouter()
	user := testutil.SignUpTestUser(t, router)

	id := uuid.New()
	require.NoError(t, database.Create(&docker.VolumeBackup{
		ID: id, ContainerID: "c", ContainerName: "c", MountPaths: []string{"/data"},
		StorageID: uuid.New(), Status: docker.BackupStatusCompleted, FileName: "x.tar.gz",
		CreatedAt: time.Now().UTC(),
	}).Error)

	testutil.MakeDelete(t, router, "/api/v1/docker/backups/"+id.String(), user.Token, http.StatusOK)

	var response backupsResponse
	testutil.MakeGetAndUnmarshal(t, router, "/api/v1/docker/backups", user.Token, http.StatusOK, &response)
	assert.Empty(t, response.Backups)
}

type configsResponse struct {
	Configs []docker.VolumeBackupConfig `json:"configs"`
}

func Test_SaveConfig_ComputesNextRun_ThenListedAndDeleted(t *testing.T) {
	testutil.SetupDb(t)
	router := testutil.NewRouter()
	user := testutil.SignUpTestUser(t, router)

	var saved docker.VolumeBackupConfig
	testutil.MakePostAndUnmarshal(t, router, "/api/v1/docker/configs", user.Token, docker.VolumeBackupConfig{
		ContainerName: "minio",
		MountPaths:    []string{"/data"},
		StorageID:     uuid.New(),
		Interval:      docker.BackupIntervalDaily,
		TimeOfDay:     "04:00",
		RetentionDays: 30,
		IsEnabled:     true,
	}, http.StatusOK, &saved)
	assert.NotEqual(t, uuid.Nil, saved.ID)
	require.NotNil(t, saved.NextRunAt)

	var listed configsResponse
	testutil.MakeGetAndUnmarshal(t, router, "/api/v1/docker/configs", user.Token, http.StatusOK, &listed)
	require.Len(t, listed.Configs, 1)
	assert.Equal(t, "minio", listed.Configs[0].ContainerName)

	testutil.MakeDelete(t, router, "/api/v1/docker/configs/"+saved.ID.String(), user.Token, http.StatusOK)

	var afterDelete configsResponse
	testutil.MakeGetAndUnmarshal(t, router, "/api/v1/docker/configs", user.Token, http.StatusOK, &afterDelete)
	assert.Empty(t, afterDelete.Configs)
}

func Test_SaveConfig_SameContainerTwice_UpsertsSingleConfig(t *testing.T) {
	testutil.SetupDb(t)
	router := testutil.NewRouter()
	user := testutil.SignUpTestUser(t, router)

	newConfig := func(retentionDays int) docker.VolumeBackupConfig {
		return docker.VolumeBackupConfig{
			ContainerName: "minio",
			MountPaths:    []string{"/data"},
			StorageID:     uuid.New(),
			Interval:      docker.BackupIntervalDaily,
			TimeOfDay:     "04:00",
			RetentionDays: retentionDays,
			IsEnabled:     true,
		}
	}
	testutil.MakePostAndUnmarshal(t, router, "/api/v1/docker/configs", user.Token, newConfig(7), http.StatusOK, nil)
	testutil.MakePostAndUnmarshal(t, router, "/api/v1/docker/configs", user.Token, newConfig(30), http.StatusOK, nil)

	var listed configsResponse
	testutil.MakeGetAndUnmarshal(t, router, "/api/v1/docker/configs", user.Token, http.StatusOK, &listed)
	require.Len(t, listed.Configs, 1)
	assert.Equal(t, 30, listed.Configs[0].RetentionDays)
}
