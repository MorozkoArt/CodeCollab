package code_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
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

func newMockRepo(t *testing.T) (code.CodeRepository, pgxmock.PgxConnIface) {
	t.Helper()

	db, mock := repotest.New(t)
	return code.NewCodeRepository(db, squirrel.StatementBuilder), mock
}

func TestCodeRepository_Upsert(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	ttl, cooldown := 10*time.Minute, time.Minute
	hash := []byte("code-hash")

	const upsertQuery = `INSERT INTO email_codes \(user_id,purpose,code_hash,expires_at,created_at\) VALUES \(\$1,\$2,\$3,\$4,\$5\) ON CONFLICT \(user_id, purpose\) DO UPDATE SET`

	t.Run("stores code", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectExec(upsertQuery).
			WithArgs(testUserID, testPurpose, hash, now.Add(ttl), now, now.Add(-cooldown)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		require.NoError(t, r.Upsert(ctx, testUserID, testPurpose, hash, now, ttl, cooldown))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("cooldown: no rows affected", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectExec(upsertQuery).
			WithArgs(testUserID, testPurpose, hash, now.Add(ttl), now, now.Add(-cooldown)).
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		err := r.Upsert(ctx, testUserID, testPurpose, hash, now, ttl, cooldown)
		assert.ErrorIs(t, err, code.ErrCooldown)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error is not ErrCooldown", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectExec(upsertQuery).
			WithArgs(testUserID, testPurpose, hash, now.Add(ttl), now, now.Add(-cooldown)).
			WillReturnError(errors.New("connection lost"))

		err := r.Upsert(ctx, testUserID, testPurpose, hash, now, ttl, cooldown)
		require.Error(t, err)
		assert.NotErrorIs(t, err, code.ErrCooldown)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCodeRepository_Attempt(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	const maxAttempts = 5

	// squirrel сортирует ключи в Eq по алфавиту: purpose, user_id
	const attemptQuery = `UPDATE email_codes SET attempts = attempts \+ 1 WHERE .+ RETURNING code_hash`

	t.Run("returns hash and increments attempts", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(attemptQuery).
			WithArgs(testPurpose, testUserID, now, maxAttempts).
			WillReturnRows(pgxmock.NewRows([]string{"code_hash"}).AddRow([]byte("stored-hash")))

		got, err := r.Attempt(ctx, testUserID, testPurpose, now, maxAttempts)
		require.NoError(t, err)
		assert.Equal(t, []byte("stored-hash"), got)
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
