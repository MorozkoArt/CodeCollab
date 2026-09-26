package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/pkg/password"
	"github.com/MorozkoArt/CodeCollab/pkg/response"
	v1 "github.com/MorozkoArt/CodeCollab/services/auth/internal/api/http/v1"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/repo"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testJWTSecret = "test_secret"
	testJWTExpiry = time.Hour

	routeRegister = "/api/v1/auth/register"
	routeLogin    = "/api/v1/auth/login"

	testEmail    = "test@example.com"
	testUsername = "testuser"
	testPassword = "password123"
	testUserID   = int64(1)
)

func newTestRouter(mockRepo *repo.MockUserRepository) http.Handler {
	jwtSvc := jwt.NewService(testJWTSecret, testJWTExpiry)
	r := chi.NewRouter()
	v1.Register(r, services.NewAuthService(mockRepo, jwtSvc), jwtSvc)
	return r
}

func postRequest(t *testing.T, h http.Handler, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := new(repo.MockUserRepository)
		mockRepo.On("ExistsByEmail", mock.Anything, "new@example.com").Return(false, nil)
		mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		body, err := json.Marshal(domain.RegisterRequest{
			Email:    "new@example.com",
			Password: testPassword,
			Username: "newuser",
		})
		require.NoError(t, err)

		w := postRequest(t, newTestRouter(mockRepo), routeRegister, body)
		assert.Equal(t, http.StatusCreated, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user exists: 409", func(t *testing.T) {
		mockRepo := new(repo.MockUserRepository)
		mockRepo.On("ExistsByEmail", mock.Anything, "exists@example.com").Return(true, nil)

		body, err := json.Marshal(domain.RegisterRequest{
			Email:    "exists@example.com",
			Password: testPassword,
			Username: "user",
		})
		require.NoError(t, err)

		w := postRequest(t, newTestRouter(mockRepo), routeRegister, body)
		assert.Equal(t, http.StatusConflict, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid body: 400", func(t *testing.T) {
		mockRepo := new(repo.MockUserRepository)
		w := postRequest(t, newTestRouter(mockRepo), routeRegister, []byte("bad json"))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		hashed, err := password.Hash(testPassword)
		require.NoError(t, err)

		mockRepo := new(repo.MockUserRepository)
		mockRepo.On("GetByEmail", mock.Anything, testEmail).Return(&domain.User{
			ID:       testUserID,
			Email:    testEmail,
			Username: testUsername,
			Password: hashed,
		}, nil)

		body, err := json.Marshal(domain.LoginRequest{
			Email:    testEmail,
			Password: testPassword,
		})
		require.NoError(t, err)

		w := postRequest(t, newTestRouter(mockRepo), routeLogin, body)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp response.Response
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.True(t, resp.Success)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong password: 401", func(t *testing.T) {
		hashed, err := password.Hash("correct_password")
		require.NoError(t, err)

		mockRepo := new(repo.MockUserRepository)
		mockRepo.On("GetByEmail", mock.Anything, testEmail).Return(&domain.User{
			ID:       testUserID,
			Email:    testEmail,
			Password: hashed,
		}, nil)

		body, err := json.Marshal(domain.LoginRequest{
			Email:    testEmail,
			Password: "wrong_password",
		})
		require.NoError(t, err)

		w := postRequest(t, newTestRouter(mockRepo), routeLogin, body)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockRepo.AssertExpectations(t)
	})
}
