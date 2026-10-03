package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler() (*Handler, *TokenManager) {
	users := newFakeUserService()
	tokens := newFakeRefreshTokenRepository()
	tm := NewTokenManager(testJWTSecret, 15*time.Minute)
	svc := NewService(users, tokens, tm, 30*24*time.Hour)
	return NewHandler(svc), tm
}

func doJSON(t *testing.T, handler http.HandlerFunc, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func TestHandler_RegisterAndLogin(t *testing.T) {
	h, _ := newTestHandler()

	rec := doJSON(t, h.Register, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Diego", "email": "diego@example.com", "password": "supersecret",
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = doJSON(t, h.Login, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"login": "diego@example.com", "password": "supersecret",
	})
	require.Equal(t, http.StatusOK, rec.Code)

	var tokens tokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tokens))
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestHandler_Register_InvalidInput(t *testing.T) {
	h, _ := newTestHandler()

	rec := doJSON(t, h.Register, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Diego", "email": "not-an-email", "password": "short",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestHandler_Register_DuplicateEmail(t *testing.T) {
	h, _ := newTestHandler()
	payload := map[string]string{"name": "Diego", "email": "diego@example.com", "password": "supersecret"}

	rec := doJSON(t, h.Register, http.MethodPost, "/api/v1/auth/register", payload)
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = doJSON(t, h.Register, http.MethodPost, "/api/v1/auth/register", payload)
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "email_taken")
}

func TestHandler_Login_EmbedsDeviceTokenInAccessToken(t *testing.T) {
	h, tm := newTestHandler()

	doJSON(t, h.Register, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Diego", "email": "diego@example.com", "password": "supersecret",
	})

	rec := doJSON(t, h.Login, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"login": "diego@example.com", "password": "supersecret", "device_token": "device-abc",
	})
	require.Equal(t, http.StatusOK, rec.Code)

	var tokens tokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tokens))

	_, deviceToken, err := tm.Parse(tokens.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "device-abc", deviceToken)
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	h, _ := newTestHandler()

	rec := doJSON(t, h.Login, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"login": "missing@example.com", "password": "whatever",
	})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_RefreshAndLogout(t *testing.T) {
	h, _ := newTestHandler()

	doJSON(t, h.Register, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Diego", "email": "diego@example.com", "password": "supersecret",
	})

	loginRec := doJSON(t, h.Login, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"login": "diego@example.com", "password": "supersecret",
	})
	var tokens tokenResponse
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &tokens))

	refreshRec := doJSON(t, h.Refresh, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	})
	require.Equal(t, http.StatusOK, refreshRec.Code)

	var rotated tokenResponse
	require.NoError(t, json.Unmarshal(refreshRec.Body.Bytes(), &rotated))
	assert.NotEqual(t, tokens.RefreshToken, rotated.RefreshToken)

	logoutRec := doJSON(t, h.Logout, http.MethodPost, "/api/v1/auth/logout", map[string]string{
		"refresh_token": rotated.RefreshToken,
	})
	assert.Equal(t, http.StatusNoContent, logoutRec.Code)
}

func TestHandler_Me_RequiresAuth(t *testing.T) {
	h, tm := newTestHandler()
	protected := RequireAuth(tm)(http.HandlerFunc(h.Me))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Me_WithValidToken(t *testing.T) {
	h, tm := newTestHandler()

	registerRec := doJSON(t, h.Register, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Diego", "email": "diego@example.com", "password": "supersecret",
	})
	var registered userResponse
	require.NoError(t, json.Unmarshal(registerRec.Body.Bytes(), &registered))

	loginRec := doJSON(t, h.Login, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"login": "diego@example.com", "password": "supersecret",
	})
	var tokens tokenResponse
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &tokens))

	protected := RequireAuth(tm)(http.HandlerFunc(h.Me))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var me userResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &me))
	assert.Equal(t, registered.ID, me.ID)
	assert.Equal(t, "diego@example.com", me.Email)
}
