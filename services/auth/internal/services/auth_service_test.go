package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/pkg/password"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/repo"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testJWTSecret = "test_secret"
	testJWTExpiry = time.Hour

	testEmail    = "test@example.com"
	testUsername = "testuser"
	testPassword = "password123"
	testUserID   = int64(1)

	existingEmail = "existing@example.com"
	existingUser  = "existinguser"
	notFoundEmail = "notfound@example.com"
)

func newTestService(mockRepo *repo.MockUserRepository) services.AuthService {
	return services.NewAuthService(mockRepo, jwt.NewService(testJWTSecret, testJWTExpiry))
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(repo.MockUserRepository)
		mockRepo.On("ExistsByEmail", ctx, testEmail).Return(false, nil)
		mockRepo.On("Create", ctx, mock.Anything).Return(nil)

		err := newTestService(mockRepo).Register(ctx, services.RegisterInput{
			Email:    testEmail,
			Password: testPassword,
			Username: testUsername,
		})
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user already exists", func(t *testing.T) {
		mockRepo := new(repo.MockUserRepository)
		mockRepo.On("ExistsByEmail", ctx, existingEmail).Return(true, nil)

		err := newTestService(mockRepo).Register(ctx, services.RegisterInput{
			Email:    existingEmail,
			Password: testPassword,
			Username: existingUser,
		})
		assert.ErrorIs(t, err, services.ErrUserExists)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		hashed, err := password.Hash(testPassword)
		require.NoError(t, err)

		mockRepo := new(repo.MockUserRepository)
		mockRepo.On("GetByEmail", ctx, testEmail).Return(&domain.User{
			ID:       testUserID,
			Email:    testEmail,
			Username: testUsername,
			Password: hashed,
		}, nil)

		token, err := newTestService(mockRepo).Login(ctx, testEmail, testPassword)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := new(repo.MockUserRepository)
		mockRepo.On("GetByEmail", ctx, notFoundEmail).Return(nil, repo.ErrUserNotFound)

		_, err := newTestService(mockRepo).Login(ctx, notFoundEmail, testPassword)
		assert.ErrorIs(t, err, services.ErrInvalidPassword)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong password", func(t *testing.T) {
		hashed, err := password.Hash("correct_password")
		require.NoError(t, err)

		mockRepo := new(repo.MockUserRepository)
		mockRepo.On("GetByEmail", ctx, testEmail).Return(&domain.User{
			ID:       testUserID,
			Email:    testEmail,
			Password: hashed,
		}, nil)

		_, err = newTestService(mockRepo).Login(ctx, testEmail, "wrong_password")
		assert.ErrorIs(t, err, services.ErrInvalidPassword)
		mockRepo.AssertExpectations(t)
	})
}
