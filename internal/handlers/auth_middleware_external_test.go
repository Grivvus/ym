package handlers_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/handlers"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_PublicPathSkipsAuthorization(t *testing.T) {
	t.Parallel()

	var called bool
	middleware := handlers.AuthMiddleware(testLogger(), []byte("secret"))
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/register", nil)
	recorder := httptest.NewRecorder()

	middleware(next).ServeHTTP(recorder, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestAuthMiddleware_OptionsRequestSkipsAuthorization(t *testing.T) {
	t.Parallel()

	var called bool
	middleware := handlers.AuthMiddleware(testLogger(), []byte("secret"))
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodOptions, "/users/42", nil)
	recorder := httptest.NewRecorder()

	middleware(next).ServeHTTP(recorder, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestAuthMiddleware_MissingBearerTokenReturnsUnauthorized(t *testing.T) {
	t.Parallel()

	var called bool
	middleware := handlers.AuthMiddleware(testLogger(), []byte("secret"))
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	recorder := httptest.NewRecorder()

	middleware(next).ServeHTTP(recorder, req)

	assert.False(t, called)
	assertErrorResponse(t, recorder, http.StatusUnauthorized, "missing bearer token")
}

func TestAuthMiddleware_InvalidAccessTokenReturnsUnauthorized(t *testing.T) {
	t.Parallel()

	var called bool
	middleware := handlers.AuthMiddleware(testLogger(), []byte("secret"))
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	recorder := httptest.NewRecorder()

	middleware(next).ServeHTTP(recorder, req)

	assert.False(t, called)
	assertErrorResponse(t, recorder, http.StatusUnauthorized, "invalid access token")
}

func TestAuthMiddleware_RefreshTokenIsRejected(t *testing.T) {
	t.Parallel()

	_, refreshToken, err := utils.CreateTokens(42, []byte("secret"))
	require.NoError(t, err)

	var called bool
	middleware := handlers.AuthMiddleware(testLogger(), []byte("secret"))
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	recorder := httptest.NewRecorder()

	middleware(next).ServeHTTP(recorder, req)

	assert.False(t, called)
	assertErrorResponse(t, recorder, http.StatusUnauthorized, "invalid access token")
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	assert.Equal(t, expectedStatus, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var body api.ErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, expectedMessage, body.Error)
}
