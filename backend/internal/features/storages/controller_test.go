package storages_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dockvol-backend/internal/features/storages"
	local_storage "dockvol-backend/internal/features/storages/models/local"
	"dockvol-backend/internal/util/testutil"
)

func Test_SaveStorage_TwoLocalStoragesInWorkspace_BothSucceed(t *testing.T) {
	testutil.SetupDb(t)
	router := testutil.NewRouter()
	user := testutil.SignUpTestUser(t, router)
	workspace := testutil.CreateTestWorkspace(t, router, user.Token)

	first := storages.Storage{
		WorkspaceID:  workspace.ID,
		Type:         storages.StorageTypeLocal,
		Name:         "first",
		LocalStorage: &local_storage.LocalStorage{},
	}
	testutil.MakePostAndUnmarshal(t, router, "/api/v1/storages", user.Token, first, http.StatusOK, nil)

	second := storages.Storage{
		WorkspaceID:  workspace.ID,
		Type:         storages.StorageTypeLocal,
		Name:         "second",
		LocalStorage: &local_storage.LocalStorage{},
	}
	testutil.MakePostAndUnmarshal(t, router, "/api/v1/storages", user.Token, second, http.StatusOK, nil)

	var listed []storages.Storage
	testutil.MakeGetAndUnmarshal(
		t, router,
		"/api/v1/storages?workspace_id="+workspace.ID.String(),
		user.Token, http.StatusOK, &listed,
	)

	require.Len(t, listed, 2)
	assert.NotEqual(t, listed[0].ID, listed[1].ID)
}

func Test_DeleteStorage_RemovesItFromTheWorkspace(t *testing.T) {
	testutil.SetupDb(t)
	router := testutil.NewRouter()
	user := testutil.SignUpTestUser(t, router)
	workspace := testutil.CreateTestWorkspace(t, router, user.Token)

	created := storages.Storage{
		WorkspaceID:  workspace.ID,
		Type:         storages.StorageTypeLocal,
		Name:         "to-delete",
		LocalStorage: &local_storage.LocalStorage{},
	}
	var saved storages.Storage
	testutil.MakePostAndUnmarshal(t, router, "/api/v1/storages", user.Token, created, http.StatusOK, &saved)

	testutil.MakeDelete(t, router, "/api/v1/storages/"+saved.ID.String(), user.Token, http.StatusOK)

	var listed []storages.Storage
	testutil.MakeGetAndUnmarshal(
		t, router,
		"/api/v1/storages?workspace_id="+workspace.ID.String(),
		user.Token, http.StatusOK, &listed,
	)
	assert.Empty(t, listed)
}
