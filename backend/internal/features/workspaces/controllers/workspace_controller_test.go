package workspaces_controllers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	workspaces_dto "dockvol-backend/internal/features/workspaces/dto"
	"dockvol-backend/internal/util/testutil"
)

func Test_CreateWorkspace_AppearsInList(t *testing.T) {
	testutil.SetupDb(t)
	router := testutil.NewRouter()
	user := testutil.SignUpTestUser(t, router)

	workspace := testutil.CreateTestWorkspace(t, router, user.Token)

	var response workspaces_dto.ListWorkspacesResponseDTO
	testutil.MakeGetAndUnmarshal(t, router, "/api/v1/workspaces", user.Token, http.StatusOK, &response)

	require.Len(t, response.Workspaces, 1)
	assert.Equal(t, workspace.ID, response.Workspaces[0].ID)
}

func Test_DeleteWorkspace_RemovesIt(t *testing.T) {
	testutil.SetupDb(t)
	router := testutil.NewRouter()
	user := testutil.SignUpTestUser(t, router)

	workspace := testutil.CreateTestWorkspace(t, router, user.Token)

	testutil.MakeDelete(t, router, "/api/v1/workspaces/"+workspace.ID.String(), user.Token, http.StatusOK)

	var response workspaces_dto.ListWorkspacesResponseDTO
	testutil.MakeGetAndUnmarshal(t, router, "/api/v1/workspaces", user.Token, http.StatusOK, &response)

	assert.Empty(t, response.Workspaces)
}
