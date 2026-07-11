package testutil

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	users_dto "dockvol-backend/internal/features/users/dto"
	workspaces_dto "dockvol-backend/internal/features/workspaces/dto"
)

// SignUpTestUser registers a fresh active user via the API and returns the
// signup response, including the bearer token for protected routes.
func SignUpTestUser(t *testing.T, router *gin.Engine) users_dto.SignInResponseDTO {
	t.Helper()

	request := users_dto.SignUpRequestDTO{
		Email:    "user-" + uuid.NewString() + "@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	var response users_dto.SignInResponseDTO
	MakePostAndUnmarshal(t, router, "/api/v1/users/signup", "", request, http.StatusOK, &response)

	return response
}

// CreateTestWorkspace creates a workspace owned by the token's user and returns it.
func CreateTestWorkspace(t *testing.T, router *gin.Engine, token string) workspaces_dto.WorkspaceResponseDTO {
	t.Helper()

	request := workspaces_dto.CreateWorkspaceRequestDTO{Name: "Workspace " + uuid.NewString()}

	var response workspaces_dto.WorkspaceResponseDTO
	MakePostAndUnmarshal(t, router, "/api/v1/workspaces", token, request, http.StatusOK, &response)

	return response
}
