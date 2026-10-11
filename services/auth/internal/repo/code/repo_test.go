package code_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/code"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/repotest"
	"github.com/jackc/pgx/v4"
	"github.com/pashagolub/pgxmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUserID  = int64(1)
	testPurpose = "login"
)

func newMockRepo(t *testing.T) (code.Repository, pgxmock.PgxConnIface) {
	t.Helper()

	db, mock := repotest.New(t)
	return code.NewRepository(db, squirrel.StatementBuilder), mock
}

func TestCodeRepository_Upsert(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	ttl, cooldown := 10*time.Minute, time.Minute
	ec := &domain.EmailCode{
		UserID: testUserID, Purpose: testPurpose, CodeHash: []byte("code-hash"),
		ExpiresAt: now.Add(ttl), CreatedAt: now,
	}
	args := []any{testUserID, testPurpose, ec.CodeHash, ec.ExpiresAt, now, now.Add(-cooldown)}

	const upsertQuery = `INSERT INTO email_codes \(user_id,purpose,code_hash,expires_at,created_at\) VALUES \(\$1,\$2,\$3,\$4,\$5\) ON CONFLICT \(user_id, purpose\) DO UPDATE SET`

	t.Run("stores code", func(t *testing.T) {
		r, mock := newMockRepo(t)
		mock.ExpectExec(upsertQuery).WithArgs(args...).WillReturnResult(pgxmock.NewResult("INSERT", 1))

		require.NoError(t, r.Upsert(ctx, ec, cooldown))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("cooldown: no rows affected", func(t *testing.T) {
		r, mock := newMockRepo(t)
		mock.ExpectExec(upsertQuery).WithArgs(args...).WillReturnResult(pgxmock.NewResult("INSERT", 0))

		assert.ErrorIs(t, r.Upsert(ctx, ec, cooldown), code.ErrCooldown)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error is not ErrCooldown", func(t *testing.T) {
		r, mock := newMockRepo(t)
		mock.ExpectExec(upsertQuery).WithArgs(args...).WillReturnError(errors.New("connection lost"))

		err := r.Upsert(ctx, ec, cooldown)
		require.Error(t, err)
		assert.NotErrorIs(t, err, code.ErrCooldown)
	})
}

func TestCodeRepository_Attempt(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	const maxAttempts = 5

	const attemptQuery = `UPDATE email_codes SET attempts = attempts \+ 1 WHERE .+ RETURNING user_id, purpose, code_hash, attempts, expires_at, created_at`

	t.Run("returns code and increments attempts", func(t *testing.T) {
		r, mock := newMockRepo(t)

		expires := now.Add(10 * time.Minute)
		cols := []string{"user_id", "purpose", "code_hash", "attempts", "expires_at", "created_at"}
		mock.ExpectQuery(attemptQuery).
			WithArgs(testPurpose, testUserID, now, maxAttempts).
			WillReturnRows(pgxmock.NewRows(cols).
				AddRow(testUserID, testPurpose, []byte("stored-hash"), 1, expires, now))

		got, err := r.Attempt(ctx, testUserID, testPurpose, now, maxAttempts)
		require.NoError(t, err)
		assert.Equal(t, testUserID, got.UserID)
		assert.Equal(t, testPurpose, got.Purpose)
		assert.Equal(t, []byte("stored-hash"), got.CodeHash)
		assert.Equal(t, 1, got.Attempts)
		assert.True(t, expires.Equal(got.ExpiresAt))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no row: missing, expired or attempts exhausted", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(attemptQuery).
			WithArgs(testPurpose, testUserID, now, maxAttempts).
			WillReturnError(pgx.ErrNoRows)

		_, err := r.Attempt(ctx, testUserID, testPurpose, now, maxAttempts)
		assert.ErrorIs(t, err, code.ErrCodeNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error is not ErrCodeNotFound", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(attemptQuery).
			WithArgs(testPurpose, testUserID, now, maxAttempts).
			WillReturnError(errors.New("connection lost"))

		_, err := r.Attempt(ctx, testUserID, testPurpose, now, maxAttempts)
		require.Error(t, err)
		assert.NotErrorIs(t, err, code.ErrCodeNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCodeRepository_Delete(t *testing.T) {
	ctx := context.Background()
	const deleteQuery = `DELETE FROM email_codes WHERE .+`

	t.Run("success", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectExec(deleteQuery).
			WithArgs(testPurpose, testUserID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		require.NoError(t, r.Delete(ctx, testUserID, testPurpose))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectExec(deleteQuery).
			WithArgs(testPurpose, testUserID).
			WillReturnError(errors.New("connection lost"))

		require.Error(t, r.Delete(ctx, testUserID, testPurpose))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
