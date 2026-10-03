package services_test

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
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/otp"
	codeRepo "github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/code"
	userRepo "github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/user"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/services"
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

var testOTPSecret = []byte("test_otp_secret_test_otp_secret_")

var testPasswordHash = sync.OnceValue(func() string {
	h, err := password.Hash(testPassword)
	if err != nil {
		panic(err)
	}
	return h
})

type env struct {
	users  *userRepo.MockUserRepository
	codes  *codeRepo.MockCodeRepository
	mailer *services.MockMailer
	jwtSvc jwt.Service
}

func newEnv() *env {
	return &env{
		users:  new(userRepo.MockUserRepository),
		codes:  new(codeRepo.MockCodeRepository),
		mailer: new(services.MockMailer),
		jwtSvc: jwt.NewService(testJWTSecret, testJWTExpiry),
	}
}

func (e *env) svc() services.AuthService {
	return services.NewAuthService(e.users, e.codes, e.jwtSvc, e.mailer, testOTPSecret)
}

func (e *env) assertExpectations(t *testing.T) {
	t.Helper()
	e.users.AssertExpectations(t)
	e.codes.AssertExpectations(t)
	e.mailer.AssertExpectations(t)
}

func (e *env) expectUpsert(purpose string, err error) *mock.Call {
	return e.codes.On("Upsert", mock.Anything, testUserID, purpose,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(err)
}

func (e *env) expectSend(purpose string, err error) *mock.Call {
	return e.mailer.On("SendCode", mock.Anything, testEmail, purpose, mock.Anything, mock.Anything).Return(err)
}

func (e *env) expectAttempt(purpose string, hash []byte, err error) {
	e.codes.On("Attempt", mock.Anything, testUserID, purpose, mock.Anything, mock.Anything).Return(hash, err)
}

func codeHash(purpose, code string) []byte {
	return otp.Hash(testOTPSecret, testUserID, purpose, code)
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
	in := services.RegisterInput{Email: testEmail, Password: testPassword, Username: testUsername}

	t.Run("new user: creates, stores code hash and sends the same code", func(t *testing.T) {
		e := newEnv()
		var storedHash []byte
		var sentCode string

		e.users.On("GetByEmail", ctx, testEmail).Return(nil, userRepo.ErrUserNotFound)
		e.users.On("Create", ctx, mock.MatchedBy(func(u *domain.User) bool {
			return u.Email == testEmail && u.Username == testUsername &&
				strings.HasPrefix(u.Password, "$argon2id$")
		})).Run(func(args mock.Arguments) {
			args.Get(1).(*domain.User).ID = testUserID
		}).Return(nil)
		e.expectUpsert(enum.PurposeRegister, nil).Run(func(args mock.Arguments) {
			storedHash = args.Get(3).([]byte)
		})
		e.expectSend(enum.PurposeRegister, nil).Run(func(args mock.Arguments) {
			sentCode = args.String(3)
		})

		require.NoError(t, e.svc().Register(ctx, in))

		assert.Len(t, sentCode, 6)
		assert.True(t, otp.Equal(storedHash, codeHash(enum.PurposeRegister, sentCode)))
		e.assertExpectations(t)
	})

	t.Run("already verified: ErrUserExists, nothing sent", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)

		err := e.svc().Register(ctx, in)
		assert.ErrorIs(t, err, services.ErrUserExists)
		e.users.AssertNumberOfCalls(t, "Create", 0)
		e.codes.AssertNumberOfCalls(t, "Upsert", 0)
		e.mailer.AssertNumberOfCalls(t, "SendCode", 0)
		e.assertExpectations(t)
	})

	t.Run("unverified exists: only resends code, does not touch the account", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectUpsert(enum.PurposeRegister, nil)
		e.expectSend(enum.PurposeRegister, nil)

		require.NoError(t, e.svc().Register(ctx, in))
		e.users.AssertNumberOfCalls(t, "Create", 0)
		e.assertExpectations(t)
	})

	t.Run("create race: ErrUserExists from repo is mapped", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, userRepo.ErrUserNotFound)
		e.users.On("Create", ctx, mock.Anything).Return(userRepo.ErrUserExists)

		err := e.svc().Register(ctx, in)
		assert.ErrorIs(t, err, services.ErrUserExists)
		e.assertExpectations(t)
	})

	t.Run("cooldown: ErrTooManyRequests, mail not sent", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectUpsert(enum.PurposeRegister, codeRepo.ErrCooldown)

		err := e.svc().Register(ctx, in)
		assert.ErrorIs(t, err, services.ErrTooManyRequests)
		e.mailer.AssertNumberOfCalls(t, "SendCode", 0)
		e.assertExpectations(t)
	})

	t.Run("mail failure: code is deleted and error returned", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectUpsert(enum.PurposeRegister, nil)
		e.expectSend(enum.PurposeRegister, errors.New("smtp down"))
		e.codes.On("Delete", mock.Anything, testUserID, enum.PurposeRegister).Return(nil)

		err := e.svc().Register(ctx, in)
		require.Error(t, err)
		assert.NotErrorIs(t, err, services.ErrTooManyRequests)
		e.assertExpectations(t)
	})
}

func TestAuthService_ConfirmRegistration(t *testing.T) {
	ctx := context.Background()

	t.Run("success: consumes code and marks email verified", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectAttempt(enum.PurposeRegister, codeHash(enum.PurposeRegister, testCode), nil)
		e.codes.On("Delete", mock.Anything, testUserID, enum.PurposeRegister).Return(nil)
		e.users.On("MarkEmailVerified", ctx, testUserID, mock.Anything).Return(nil)

		require.NoError(t, e.svc().ConfirmRegistration(ctx, testEmail, testCode))
		e.assertExpectations(t)
	})

	t.Run("wrong code: ErrInvalidCode, nothing consumed", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectAttempt(enum.PurposeRegister, codeHash(enum.PurposeRegister, "000000"), nil)

		err := e.svc().ConfirmRegistration(ctx, testEmail, testCode)
		assert.ErrorIs(t, err, services.ErrInvalidCode)
		e.codes.AssertNumberOfCalls(t, "Delete", 0)
		e.users.AssertNumberOfCalls(t, "MarkEmailVerified", 0)
		e.assertExpectations(t)
	})

	t.Run("login code cannot confirm registration", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectAttempt(enum.PurposeRegister, codeHash(enum.PurposeLogin, testCode), nil)

		err := e.svc().ConfirmRegistration(ctx, testEmail, testCode)
		assert.ErrorIs(t, err, services.ErrInvalidCode)
		e.users.AssertNumberOfCalls(t, "MarkEmailVerified", 0)
		e.assertExpectations(t)
	})

	t.Run("code missing, expired or attempts exhausted", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)
		e.expectAttempt(enum.PurposeRegister, nil, codeRepo.ErrCodeNotFound)

		err := e.svc().ConfirmRegistration(ctx, testEmail, testCode)
		assert.ErrorIs(t, err, services.ErrInvalidCode)
		e.assertExpectations(t)
	})

	t.Run("unknown email: ErrInvalidCode (no user enumeration)", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, userRepo.ErrUserNotFound)

		err := e.svc().ConfirmRegistration(ctx, testEmail, testCode)
		assert.ErrorIs(t, err, services.ErrInvalidCode)
		e.codes.AssertNumberOfCalls(t, "Attempt", 0)
		e.assertExpectations(t)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("success: sends login code", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)
		e.expectUpsert(enum.PurposeLogin, nil)
		e.expectSend(enum.PurposeLogin, nil)

		require.NoError(t, e.svc().Login(ctx, testEmail, testPassword))
		e.assertExpectations(t)
	})

	t.Run("user not found: ErrInvalidPassword, nothing sent", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, userRepo.ErrUserNotFound)

		err := e.svc().Login(ctx, testEmail, testPassword)
		assert.ErrorIs(t, err, services.ErrInvalidPassword)
		e.mailer.AssertNumberOfCalls(t, "SendCode", 0)
		e.assertExpectations(t)
	})

	t.Run("wrong password: ErrInvalidPassword, nothing sent", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)

		err := e.svc().Login(ctx, testEmail, "wrong_password")
		assert.ErrorIs(t, err, services.ErrInvalidPassword)
		e.codes.AssertNumberOfCalls(t, "Upsert", 0)
		e.mailer.AssertNumberOfCalls(t, "SendCode", 0)
		e.assertExpectations(t)
	})

	t.Run("email not verified: ErrEmailNotVerified, no code issued", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(unverifiedUser(), nil)

		err := e.svc().Login(ctx, testEmail, testPassword)
		assert.ErrorIs(t, err, services.ErrEmailNotVerified)
		e.codes.AssertNumberOfCalls(t, "Upsert", 0)
		e.assertExpectations(t)
	})

	t.Run("cooldown: ErrTooManyRequests", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)
		e.expectUpsert(enum.PurposeLogin, codeRepo.ErrCooldown)

		err := e.svc().Login(ctx, testEmail, testPassword)
		assert.ErrorIs(t, err, services.ErrTooManyRequests)
		e.mailer.AssertNumberOfCalls(t, "SendCode", 0)
		e.assertExpectations(t)
	})
}

func TestAuthService_VerifyLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("success: returns valid JWT for the user", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)
		e.expectAttempt(enum.PurposeLogin, codeHash(enum.PurposeLogin, testCode), nil)
		e.codes.On("Delete", mock.Anything, testUserID, enum.PurposeLogin).Return(nil)

		token, err := e.svc().VerifyLogin(ctx, testEmail, testCode)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		claims, err := e.jwtSvc.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, testUserID, claims.UserID)
		assert.Equal(t, testEmail, claims.Email)
		e.assertExpectations(t)
	})

	t.Run("wrong code: no token, code not consumed", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)
		e.expectAttempt(enum.PurposeLogin, codeHash(enum.PurposeLogin, "000000"), nil)

		token, err := e.svc().VerifyLogin(ctx, testEmail, testCode)
		assert.ErrorIs(t, err, services.ErrInvalidCode)
		assert.Empty(t, token)
		e.codes.AssertNumberOfCalls(t, "Delete", 0)
		e.assertExpectations(t)
	})

	t.Run("registration code cannot be used to log in", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)
		e.expectAttempt(enum.PurposeLogin, codeHash(enum.PurposeRegister, testCode), nil)

		token, err := e.svc().VerifyLogin(ctx, testEmail, testCode)
		assert.ErrorIs(t, err, services.ErrInvalidCode)
		assert.Empty(t, token)
		e.assertExpectations(t)
	})

	t.Run("attempts exhausted or expired", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(verifiedUser(), nil)
		e.expectAttempt(enum.PurposeLogin, nil, codeRepo.ErrCodeNotFound)

		_, err := e.svc().VerifyLogin(ctx, testEmail, testCode)
		assert.ErrorIs(t, err, services.ErrInvalidCode)
		e.assertExpectations(t)
	})

	t.Run("unknown email: ErrInvalidCode", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", ctx, testEmail).Return(nil, userRepo.ErrUserNotFound)

		_, err := e.svc().VerifyLogin(ctx, testEmail, testCode)
		assert.ErrorIs(t, err, services.ErrInvalidCode)
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
		other := jwt.NewService("another_secret", testJWTExpiry)
		token, err := other.GenerateToken(testUserID, testEmail)
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
