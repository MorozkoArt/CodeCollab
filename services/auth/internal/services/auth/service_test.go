package auth_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/pkg/password"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	userRepo "github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/user"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/services/auth"
	codesvc "github.com/MorozkoArt/CodeCollab/services/auth/internal/services/code"
	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/enum"
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
	testCode     = "123456"
)

var errDB = errors.New("db down")

var testPasswordHash = sync.OnceValue(func() string {
	h, err := password.Hash(testPassword)
	if err != nil {
		panic(err)
	}
	return h
})

type env struct {
	users  *userRepo.MockRepository
	codes  *codesvc.MockService
	jwtSvc jwt.Service
}

func newEnv() *env {
	return &env{
		users:  new(userRepo.MockRepository),
		codes:  new(codesvc.MockService),
		jwtSvc: jwt.NewService(testJWTSecret, testJWTExpiry),
	}
}

func (e *env) svc() auth.Service {
	return auth.NewService(e.users, e.codes, e.jwtSvc)
}

func (e *env) assertExpectations(t *testing.T) {
	t.Helper()
	e.users.AssertExpectations(t)
	e.codes.AssertExpectations(t)
}

func (e *env) expectIssue(purpose string, err error) {
	e.codes.On("Issue", mock.Anything, isTestUser(), purpose).Return(err)
}

func (e *env) expectVerify(purpose string, err error) {
	e.codes.On("Verify", mock.Anything, isTestUser(), purpose, testCode).Return(err)
}

func isTestUser() any {
	return mock.MatchedBy(func(u *domain.User) bool {
		return u.ID == testUserID && u.Email == testEmail
	})
}

func unverifiedUser() *domain.User {
	return &domain.User{ID: testUserID, Email: testEmail, Username: testUsername, Password: testPasswordHash()}
}

func verifiedUser() *domain.User {
	u := unverifiedUser()
	at := time.Now().Add(-time.Hour)
	u.EmailVerifiedAt = &at
	return u
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()
	in := auth.RegisterInput{Email: testEmail, Password: testPassword, Username: testUsername}

	t.Run("new user: created with argon2id hash, code issued", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, userRepo.ErrUserNotFound)
		e.users.On("Create", ctx, mock.MatchedBy(func(u *domain.User) bool {
			return u.Email == testEmail && u.Username == testUsername &&
				strings.HasPrefix(u.Password, "$argon2id$")
		})).Run(func(args mock.Arguments) {
			args.Get(1).(*domain.User).ID = testUserID
		}).Return(nil)
		e.expectIssue(enum.PurposeRegister, nil)

		require.NoError(t, e.svc().Register(ctx, in))
		e.assertExpectations(t)
	})

	t.Run("already verified: ErrUserExists, nothing created or issued", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)

		assert.ErrorIs(t, e.svc().Register(ctx, in), auth.ErrUserExists)
		e.users.AssertNumberOfCalls(t, "Create", 0)
		e.codes.AssertNumberOfCalls(t, "Issue", 0)
		e.assertExpectations(t)
	})

	t.Run("unverified exists: only resends code, account untouched", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectIssue(enum.PurposeRegister, nil)

		require.NoError(t, e.svc().Register(ctx, in))
		e.users.AssertNumberOfCalls(t, "Create", 0)
		e.assertExpectations(t)
	})

	t.Run("create race: repo ErrUserExists is mapped", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, userRepo.ErrUserNotFound)
		e.users.On("Create", ctx, mock.Anything).Return(userRepo.ErrUserExists)

		assert.ErrorIs(t, e.svc().Register(ctx, in), auth.ErrUserExists)
		e.codes.AssertNumberOfCalls(t, "Issue", 0)
		e.assertExpectations(t)
	})

	t.Run("create db error: wrapped, code not issued", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, userRepo.ErrUserNotFound)
		e.users.On("Create", ctx, mock.Anything).Return(errDB)

		err := e.svc().Register(ctx, in)
		assert.ErrorIs(t, err, errDB)
		assert.NotErrorIs(t, err, auth.ErrUserExists)
		e.codes.AssertNumberOfCalls(t, "Issue", 0)
	})

	t.Run("lookup db error: returned, nothing else called", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, errDB)

		assert.ErrorIs(t, e.svc().Register(ctx, in), errDB)
		e.users.AssertNumberOfCalls(t, "Create", 0)
		e.codes.AssertNumberOfCalls(t, "Issue", 0)
	})

	t.Run("code service error is propagated", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectIssue(enum.PurposeRegister, codesvc.ErrTooManyRequests)

		assert.ErrorIs(t, e.svc().Register(ctx, in), codesvc.ErrTooManyRequests)
		e.assertExpectations(t)
	})
}

func TestAuthService_ConfirmRegistration(t *testing.T) {
	ctx := context.Background()

	t.Run("success: code verified, email marked verified", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectVerify(enum.PurposeRegister, nil)
		e.users.On("MarkEmailVerified", ctx, testUserID, mock.Anything).Return(nil)

		require.NoError(t, e.svc().ConfirmRegistration(ctx, testEmail, testCode))
		e.assertExpectations(t)
	})

	t.Run("code rejected: ErrInvalidCode, email stays unverified", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectVerify(enum.PurposeRegister, codesvc.ErrInvalidCode)

		assert.ErrorIs(t, e.svc().ConfirmRegistration(ctx, testEmail, testCode), codesvc.ErrInvalidCode)
		e.users.AssertNumberOfCalls(t, "MarkEmailVerified", 0)
		e.assertExpectations(t)
	})

	t.Run("unknown email: ErrInvalidCode, code not checked", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, userRepo.ErrUserNotFound)

		assert.ErrorIs(t, e.svc().ConfirmRegistration(ctx, testEmail, testCode), codesvc.ErrInvalidCode)
		e.codes.AssertNumberOfCalls(t, "Verify", 0)
		e.assertExpectations(t)
	})

	t.Run("mark verified fails: error returned", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectVerify(enum.PurposeRegister, nil)
		e.users.On("MarkEmailVerified", ctx, testUserID, mock.Anything).Return(errDB)

		assert.ErrorIs(t, e.svc().ConfirmRegistration(ctx, testEmail, testCode), errDB)
	})

	t.Run("lookup db error is not masked as invalid code", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, errDB)

		err := e.svc().ConfirmRegistration(ctx, testEmail, testCode)
		assert.ErrorIs(t, err, errDB)
		assert.NotErrorIs(t, err, codesvc.ErrInvalidCode)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("success: login code issued", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)
		e.expectIssue(enum.PurposeLogin, nil)

		require.NoError(t, e.svc().Login(ctx, testEmail, testPassword))
		e.assertExpectations(t)
	})

	t.Run("user not found: ErrInvalidPassword, no code", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, userRepo.ErrUserNotFound)

		assert.ErrorIs(t, e.svc().Login(ctx, testEmail, testPassword), auth.ErrInvalidPassword)
		e.codes.AssertNumberOfCalls(t, "Issue", 0)
		e.assertExpectations(t)
	})

	t.Run("wrong password: ErrInvalidPassword, no code", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)

		assert.ErrorIs(t, e.svc().Login(ctx, testEmail, "wrong_password"), auth.ErrInvalidPassword)
		e.codes.AssertNumberOfCalls(t, "Issue", 0)
		e.assertExpectations(t)
	})

	t.Run("email not verified: ErrEmailNotVerified, no code", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)

		assert.ErrorIs(t, e.svc().Login(ctx, testEmail, testPassword), auth.ErrEmailNotVerified)
		e.codes.AssertNumberOfCalls(t, "Issue", 0)
		e.assertExpectations(t)
	})

	t.Run("wrong password is checked before verification status", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)

		err := e.svc().Login(ctx, testEmail, "wrong_password")
		assert.ErrorIs(t, err, auth.ErrInvalidPassword)
		assert.NotErrorIs(t, err, auth.ErrEmailNotVerified)
	})

	t.Run("code service error is propagated", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)
		e.expectIssue(enum.PurposeLogin, codesvc.ErrTooManyRequests)

		assert.ErrorIs(t, e.svc().Login(ctx, testEmail, testPassword), codesvc.ErrTooManyRequests)
		e.assertExpectations(t)
	})

	t.Run("lookup db error is not masked as invalid password", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, errDB)

		err := e.svc().Login(ctx, testEmail, testPassword)
		assert.ErrorIs(t, err, errDB)
		assert.NotErrorIs(t, err, auth.ErrInvalidPassword)
	})
}

func TestAuthService_VerifyLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("success: valid JWT for the user", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)
		e.expectVerify(enum.PurposeLogin, nil)

		token, err := e.svc().VerifyLogin(ctx, testEmail, testCode)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		claims, err := e.jwtSvc.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, testUserID, claims.UserID)
		assert.Equal(t, testEmail, claims.Email)
		e.assertExpectations(t)
	})

	t.Run("code rejected: no token", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)
		e.expectVerify(enum.PurposeLogin, codesvc.ErrInvalidCode)

		token, err := e.svc().VerifyLogin(ctx, testEmail, testCode)
		assert.ErrorIs(t, err, codesvc.ErrInvalidCode)
		assert.Empty(t, token)
		e.assertExpectations(t)
	})

	t.Run("unknown email: ErrInvalidCode, code not checked", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, userRepo.ErrUserNotFound)

		_, err := e.svc().VerifyLogin(ctx, testEmail, testCode)
		assert.ErrorIs(t, err, codesvc.ErrInvalidCode)
		e.codes.AssertNumberOfCalls(t, "Verify", 0)
		e.assertExpectations(t)
	})
}

func TestAuthService_Validate(t *testing.T) {
	ctx := context.Background()

	t.Run("valid token returns user id", func(t *testing.T) {
		e := newEnv()
		token, err := e.jwtSvc.GenerateToken(testUserID, testEmail)
		require.NoError(t, err)

		id, err := e.svc().Validate(ctx, token)
		require.NoError(t, err)
		assert.Equal(t, testUserID, id)
	})

	t.Run("garbage token", func(t *testing.T) {
		_, err := newEnv().svc().Validate(ctx, "not-a-jwt")
		assert.Error(t, err)
	})

	t.Run("token signed with another secret", func(t *testing.T) {
		token, err := jwt.NewService("another_secret", testJWTExpiry).GenerateToken(testUserID, testEmail)
		require.NoError(t, err)

		_, err = newEnv().svc().Validate(ctx, token)
		assert.Error(t, err)
	})
}

func TestAuthService_GetUser(t *testing.T) {
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByID", ctx, testUserID).Return(verifiedUser(), nil)

		u, err := e.svc().GetUser(ctx, testUserID)
		require.NoError(t, err)
		assert.Equal(t, testEmail, u.Email)
		e.assertExpectations(t)
	})

	t.Run("not found keeps ErrUserNotFound in chain", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByID", ctx, int64(999)).Return(nil, userRepo.ErrUserNotFound)

		_, err := e.svc().GetUser(ctx, 999)
		assert.ErrorIs(t, err, userRepo.ErrUserNotFound)
		e.assertExpectations(t)
	})
}
