package code_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/mailer"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/otp"
	coderepo "github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/code"
	codesvc "github.com/MorozkoArt/CodeCollab/services/auth/internal/services/code"
	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/enum"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testUserID  = int64(1)
	testEmail   = "test@example.com"
	testPurpose = enum.PurposeLogin
	testCode    = "123456"
)

var (
	testSecret = []byte("test_otp_secret_test_otp_secret_")
	errSMTP    = errors.New("smtp down")
	errDB      = errors.New("db down")
)

type env struct {
	repo   *coderepo.MockRepository
	mailer *mailer.MockMailer
}

func newEnv() (*env, codesvc.Service) {
	e := &env{repo: new(coderepo.MockRepository), mailer: new(mailer.MockMailer)}
	return e, codesvc.NewService(e.repo, e.mailer, testSecret)
}

func testUser() *domain.User { return &domain.User{ID: testUserID, Email: testEmail} }

func hashOf(purpose, c string) []byte { return otp.Hash(testSecret, testUserID, purpose, c) }

func TestCodeService_Issue(t *testing.T) {
	ctx := context.Background()

	t.Run("stores hash of the same code that is mailed", func(t *testing.T) {
		e, svc := newEnv()
		var stored *domain.EmailCode
		var sent string

		e.repo.On("Upsert", ctx, mock.Anything, time.Minute).Run(func(a mock.Arguments) {
			stored = a.Get(1).(*domain.EmailCode)
		}).Return(nil)
		e.mailer.On("SendCode", ctx, testEmail, testPurpose, mock.Anything, 10*time.Minute).Run(func(a mock.Arguments) {
			sent = a.String(3)
		}).Return(nil)

		require.NoError(t, svc.Issue(ctx, testUser(), testPurpose))

		assert.Regexp(t, `^\d{6}$`, sent)
		assert.Equal(t, testUserID, stored.UserID)
		assert.Equal(t, testPurpose, stored.Purpose)
		assert.True(t, otp.Equal(stored.CodeHash, hashOf(testPurpose, sent)))
		assert.Equal(t, 10*time.Minute, stored.ExpiresAt.Sub(stored.CreatedAt))
		e.repo.AssertExpectations(t)
		e.mailer.AssertExpectations(t)
	})

	t.Run("plain code is never stored", func(t *testing.T) {
		e, svc := newEnv()
		var stored *domain.EmailCode
		var sent string

		e.repo.On("Upsert", ctx, mock.Anything, mock.Anything).Run(func(a mock.Arguments) {
			stored = a.Get(1).(*domain.EmailCode)
		}).Return(nil)
		e.mailer.On("SendCode", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(a mock.Arguments) {
			sent = a.String(3)
		}).Return(nil)

		require.NoError(t, svc.Issue(ctx, testUser(), testPurpose))
		assert.NotEqual(t, []byte(sent), stored.CodeHash)
	})

	t.Run("cooldown: ErrTooManyRequests, nothing mailed", func(t *testing.T) {
		e, svc := newEnv()
		e.repo.On("Upsert", ctx, mock.Anything, mock.Anything).Return(coderepo.ErrCooldown)

		assert.ErrorIs(t, svc.Issue(ctx, testUser(), testPurpose), codesvc.ErrTooManyRequests)
		e.mailer.AssertNumberOfCalls(t, "SendCode", 0)
		e.repo.AssertNumberOfCalls(t, "Delete", 0)
	})

	t.Run("store db error: wrapped, nothing mailed", func(t *testing.T) {
		e, svc := newEnv()
		e.repo.On("Upsert", ctx, mock.Anything, mock.Anything).Return(errDB)

		err := svc.Issue(ctx, testUser(), testPurpose)
		assert.ErrorIs(t, err, errDB)
		assert.NotErrorIs(t, err, codesvc.ErrTooManyRequests)
		e.mailer.AssertNumberOfCalls(t, "SendCode", 0)
	})

	t.Run("mail failure: code is deleted, mail error returned", func(t *testing.T) {
		e, svc := newEnv()
		e.repo.On("Upsert", ctx, mock.Anything, mock.Anything).Return(nil)
		e.mailer.On("SendCode", ctx, mock.Anything, testPurpose, mock.Anything, mock.Anything).Return(errSMTP)
		e.repo.On("Delete", ctx, testUserID, testPurpose).Return(nil)

		assert.ErrorIs(t, svc.Issue(ctx, testUser(), testPurpose), errSMTP)
		e.repo.AssertExpectations(t)
	})

	t.Run("mail failure and delete failure: mail error still returned", func(t *testing.T) {
		e, svc := newEnv()
		e.repo.On("Upsert", ctx, mock.Anything, mock.Anything).Return(nil)
		e.mailer.On("SendCode", ctx, mock.Anything, testPurpose, mock.Anything, mock.Anything).Return(errSMTP)
		e.repo.On("Delete", ctx, testUserID, testPurpose).Return(errDB)

		assert.ErrorIs(t, svc.Issue(ctx, testUser(), testPurpose), errSMTP)
	})
}

func TestCodeService_Verify(t *testing.T) {
	ctx := context.Background()

	expectAttempt := func(e *env, c *domain.EmailCode, err error) {
		e.repo.On("Attempt", ctx, testUserID, testPurpose, mock.Anything, 5).Return(c, err)
	}

	t.Run("correct code is accepted and consumed", func(t *testing.T) {
		e, svc := newEnv()
		expectAttempt(e, &domain.EmailCode{CodeHash: hashOf(testPurpose, testCode)}, nil)
		e.repo.On("Delete", ctx, testUserID, testPurpose).Return(nil)

		require.NoError(t, svc.Verify(ctx, testUser(), testPurpose, testCode))
		e.repo.AssertExpectations(t)
	})

	t.Run("wrong code is rejected and not consumed", func(t *testing.T) {
		e, svc := newEnv()
		expectAttempt(e, &domain.EmailCode{CodeHash: hashOf(testPurpose, "000000")}, nil)

		assert.ErrorIs(t, svc.Verify(ctx, testUser(), testPurpose, testCode), codesvc.ErrInvalidCode)
		e.repo.AssertNumberOfCalls(t, "Delete", 0)
	})

	t.Run("code issued for another purpose is rejected", func(t *testing.T) {
		e, svc := newEnv()
		expectAttempt(e, &domain.EmailCode{CodeHash: hashOf(enum.PurposeRegister, testCode)}, nil)

		assert.ErrorIs(t, svc.Verify(ctx, testUser(), testPurpose, testCode), codesvc.ErrInvalidCode)
	})

	t.Run("code issued for another user is rejected", func(t *testing.T) {
		e, svc := newEnv()
		expectAttempt(e, &domain.EmailCode{CodeHash: otp.Hash(testSecret, 2, testPurpose, testCode)}, nil)

		assert.ErrorIs(t, svc.Verify(ctx, testUser(), testPurpose, testCode), codesvc.ErrInvalidCode)
	})

	t.Run("missing, expired or exhausted: ErrInvalidCode", func(t *testing.T) {
		e, svc := newEnv()
		expectAttempt(e, nil, coderepo.ErrCodeNotFound)

		assert.ErrorIs(t, svc.Verify(ctx, testUser(), testPurpose, testCode), codesvc.ErrInvalidCode)
		e.repo.AssertNumberOfCalls(t, "Delete", 0)
	})

	t.Run("attempt db error is not masked as invalid code", func(t *testing.T) {
		e, svc := newEnv()
		expectAttempt(e, nil, errDB)

		err := svc.Verify(ctx, testUser(), testPurpose, testCode)
		assert.ErrorIs(t, err, errDB)
		assert.NotErrorIs(t, err, codesvc.ErrInvalidCode)
	})

	t.Run("consume failure: error returned even for correct code", func(t *testing.T) {
		e, svc := newEnv()
		expectAttempt(e, &domain.EmailCode{CodeHash: hashOf(testPurpose, testCode)}, nil)
		e.repo.On("Delete", ctx, testUserID, testPurpose).Return(errDB)

		assert.ErrorIs(t, svc.Verify(ctx, testUser(), testPurpose, testCode), errDB)
	})
}
