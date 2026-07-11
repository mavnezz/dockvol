package users_controllers_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	users_dto "dockvol-backend/internal/features/users/dto"
	"dockvol-backend/internal/util/testutil"
)

func Test_SignUp_ReturnsTokenAndUserID(t *testing.T) {
	testutil.SetupDb(t)
	router := testutil.NewRouter()

	user := testutil.SignUpTestUser(t, router)

	assert.NotEmpty(t, user.Token)
	assert.NotEqual(t, uuid.Nil, user.UserID)
}

func Test_SignIn_WithValidCredentials_ReturnsToken(t *testing.T) {
	testutil.SetupDb(t)
	router := testutil.NewRouter()

	email := "signin-" + uuid.NewString() + "@example.com"
	signUp := users_dto.SignUpRequestDTO{Email: email, Password: "password123", Name: "Test"}
	testutil.MakePostAndUnmarshal(t, router, "/api/v1/users/signup", "", signUp, http.StatusOK, nil)

	var response users_dto.SignInResponseDTO
	signIn := users_dto.SignInRequestDTO{Email: email, Password: "password123"}
	testutil.MakePostAndUnmarshal(t, router, "/api/v1/users/signin", "", signIn, http.StatusOK, &response)

	assert.NotEmpty(t, response.Token)
}

func Test_SignIn_WithWrongPassword_IsRejected(t *testing.T) {
	testutil.SetupDb(t)
	router := testutil.NewRouter()

	email := "wrongpw-" + uuid.NewString() + "@example.com"
	signUp := users_dto.SignUpRequestDTO{Email: email, Password: "password123", Name: "Test"}
	testutil.MakePostAndUnmarshal(t, router, "/api/v1/users/signup", "", signUp, http.StatusOK, nil)

	recorder := testutil.MakeRequest(
		t, router, http.MethodPost, "/api/v1/users/signin", "",
		users_dto.SignInRequestDTO{Email: email, Password: "wrongpassword"},
	)

	assert.NotEqual(t, http.StatusOK, recorder.Code)
}

func Test_ProtectedRoute_WithoutToken_ReturnsUnauthorized(t *testing.T) {
	testutil.SetupDb(t)
	router := testutil.NewRouter()

	recorder := testutil.MakeRequest(t, router, http.MethodGet, "/api/v1/workspaces", "", nil)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}
