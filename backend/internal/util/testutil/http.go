package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func MakeRequest(
	t *testing.T,
	router *gin.Engine,
	method, path, token string,
	body any,
) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		marshaled, err := json.Marshal(body)
		require.NoError(t, err)
		payload = marshaled
	}

	httpRequest := httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewReader(payload))
	httpRequest.Header.Set("Content-Type", "application/json")
	if token != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	return recorder
}

func MakePostAndUnmarshal(
	t *testing.T,
	router *gin.Engine,
	path, token string,
	body any,
	expectedStatus int,
	out any,
) {
	t.Helper()

	recorder := MakeRequest(t, router, http.MethodPost, path, token, body)
	require.Equal(t, expectedStatus, recorder.Code, "response body: %s", recorder.Body.String())

	if out != nil {
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), out))
	}
}

func MakeGetAndUnmarshal(
	t *testing.T,
	router *gin.Engine,
	path, token string,
	expectedStatus int,
	out any,
) {
	t.Helper()

	recorder := MakeRequest(t, router, http.MethodGet, path, token, nil)
	require.Equal(t, expectedStatus, recorder.Code, "response body: %s", recorder.Body.String())

	if out != nil {
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), out))
	}
}

func MakeDelete(
	t *testing.T,
	router *gin.Engine,
	path, token string,
	expectedStatus int,
) {
	t.Helper()

	recorder := MakeRequest(t, router, http.MethodDelete, path, token, nil)
	require.Equal(t, expectedStatus, recorder.Code, "response body: %s", recorder.Body.String())
}
