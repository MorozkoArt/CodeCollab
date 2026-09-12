package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/MorozkoArt/CodeCollab/internal/domain"
	"github.com/MorozkoArt/CodeCollab/internal/repository"
	"github.com/MorozkoArt/CodeCollab/internal/services"
	"github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/pkg/password"
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

func newTestService(repo *repository.MockUserRepository) *services.AuthService {
	return services.NewAuthService(repo, jwt.NewService(testJWTSecret, testJWTExpiry))
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(repository.MockUserRepository)
		mockRepo.On("Create", ctx, mock.Anything).Return(nil)

		err := newTestService(mockRepo).Register(ctx, &domain.RegisterRequest{
			Email:    testEmail,
			Password: testPassword,
			Username: testUsername,
		})
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user already exists", func(t *testing.T) {
		mockRepo := new(repository.MockUserRepository)
		mockRepo.On("Create", ctx, mock.Anything).Return(repository.ErrUserExists)

		err := newTestService(mockRepo).Register(ctx, &domain.RegisterRequest{
			Email:    existingEmail,
			Password: testPassword,
			Username: existingUser,
		})
		assert.ErrorIs(t, err, repository.ErrUserExists)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		hashed, err := password.Hash(testPassword)
		require.NoError(t, err)

		mockRepo := new(repository.MockUserRepository)
		mockRepo.On("GetByEmail", ctx, testEmail).Return(&domain.User{
			ID:       testUserID,
			Email:    testEmail,
			Username: testUsername,
			Password: hashed,
		}, nil)

		user, token, err := newTestService(mockRepo).Login(ctx, &domain.LoginRequest{
			Email:    testEmail,
			Password: testPassword,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, testUserID, user.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := new(repository.MockUserRepository)
		mockRepo.On("GetByEmail", ctx, notFoundEmail).
			Return(nil, repository.ErrUserNotFound)

		_, _, err := newTestService(mockRepo).Login(ctx, &domain.LoginRequest{
			Email:    notFoundEmail,
			Password: testPassword,
		})
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong password", func(t *testing.T) {
		hashed, _ := password.Hash("correct_password")

		mockRepo := new(repository.MockUserRepository)
		mockRepo.On("GetByEmail", ctx, testEmail).Return(&domain.User{
			ID:       testUserID,
			Email:    testEmail,
			Password: hashed,
		}, nil)

		_, _, err := newTestService(mockRepo).Login(ctx, &domain.LoginRequest{
			Email:    testEmail,
			Password: "wrong_password",
		})
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}
