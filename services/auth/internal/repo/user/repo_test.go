package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/repotest"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/user"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/pashagolub/pgxmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	getByEmailQuery = `SELECT id, username, email, password, email_verified_at, created_at, updated_at FROM users WHERE email = \$1`
	getByIDQuery    = `SELECT id, username, email, created_at, updated_at FROM users WHERE id = \$1`
	uniqueViolation = "23505"
)

var getByEmailCols = []string{"id", "username", "email", "password", "email_verified_at", "created_at", "updated_at"}

func newMockRepo(t *testing.T) (user.UserRepository, pgxmock.PgxConnIface) {
	t.Helper()

	db, mock := repotest.New(t)
	return user.NewUserRepository(db, squirrel.StatementBuilder), mock
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("exists", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(`SELECT COUNT\(1\) FROM users WHERE email = \$1`).
			WithArgs("test@example.com").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

		exists, err := r.ExistsByEmail(ctx, "test@example.com")
		require.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not exists", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(`SELECT COUNT\(1\) FROM users WHERE email = \$1`).
			WithArgs("new@example.com").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

		exists, err := r.ExistsByEmail(ctx, "new@example.com")
		require.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Create(t *testing.T) {
	ctx := context.Background()
	const insertQuery = `INSERT INTO users \(username,email,password\) VALUES \(\$1,\$2,\$3\) RETURNING id`

	t.Run("creates user and fills ID", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(insertQuery).
			WithArgs("testuser", "test@example.com", "hashed_password").
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(42)))

		u := &domain.User{Username: "testuser", Email: "test@example.com", Password: "hashed_password"}
		require.NoError(t, r.Create(ctx, u))
		assert.Equal(t, int64(42), u.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("unique violation maps to ErrUserExists", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(insertQuery).
			WithArgs("testuser", "test@example.com", "hashed_password").
			WillReturnError(&pgconn.PgError{Code: uniqueViolation})

		err := r.Create(ctx, &domain.User{Username: "testuser", Email: "test@example.com", Password: "hashed_password"})
		assert.ErrorIs(t, err, user.ErrUserExists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("other db error is wrapped", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(insertQuery).
			WithArgs("testuser", "test@example.com", "hashed_password").
			WillReturnError(errors.New("connection lost"))

		err := r.Create(ctx, &domain.User{Username: "testuser", Email: "test@example.com", Password: "hashed_password"})
		require.Error(t, err)
		assert.NotErrorIs(t, err, user.ErrUserExists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("returns verified user", func(t *testing.T) {
		r, mock := newMockRepo(t)

		now := time.Now().Truncate(time.Second)
		verifiedAt := now.Add(-time.Hour)
		mock.ExpectQuery(getByEmailQuery).
			WithArgs("test@example.com").
			WillReturnRows(pgxmock.NewRows(getByEmailCols).
				AddRow(int64(1), "testuser", "test@example.com", "hashed", &verifiedAt, now, now))

		got, err := r.GetByEmail(ctx, "test@example.com")
		require.NoError(t, err)
		assert.Equal(t, int64(1), got.ID)
		assert.Equal(t, "testuser", got.Username)
		assert.Equal(t, "hashed", got.Password)
		require.NotNil(t, got.EmailVerifiedAt)
		assert.True(t, verifiedAt.Equal(*got.EmailVerifiedAt))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns unverified user with nil EmailVerifiedAt", func(t *testing.T) {
		r, mock := newMockRepo(t)

		now := time.Now().Truncate(time.Second)
		mock.ExpectQuery(getByEmailQuery).
			WithArgs("test@example.com").
			WillReturnRows(pgxmock.NewRows(getByEmailCols).
				AddRow(int64(1), "testuser", "test@example.com", "hashed", (*time.Time)(nil), now, now))

		got, err := r.GetByEmail(ctx, "test@example.com")
		require.NoError(t, err)
		assert.Nil(t, got.EmailVerifiedAt)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrUserNotFound when no rows", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(getByEmailQuery).
			WithArgs("notfound@example.com").
			WillReturnError(pgx.ErrNoRows)

		_, err := r.GetByEmail(ctx, "notfound@example.com")
		assert.ErrorIs(t, err, user.ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("other db error is not ErrUserNotFound", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(getByEmailQuery).
			WithArgs("test@example.com").
			WillReturnError(errors.New("connection lost"))

		_, err := r.GetByEmail(ctx, "test@example.com")
		require.Error(t, err)
		assert.NotErrorIs(t, err, user.ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns user when found", func(t *testing.T) {
		r, mock := newMockRepo(t)

		now := time.Now().Truncate(time.Second)
		mock.ExpectQuery(getByIDQuery).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "username", "email", "created_at", "updated_at"}).
				AddRow(int64(1), "testuser", "test@example.com", now, now))

		got, err := r.GetByID(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), got.ID)
		assert.Equal(t, "test@example.com", got.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when not found", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(getByIDQuery).
			WithArgs(int64(999999)).
			WillReturnError(pgx.ErrNoRows)

		_, err := r.GetByID(ctx, 999999)
		assert.ErrorIs(t, err, user.ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_MarkEmailVerified(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	const updateQuery = `UPDATE users SET email_verified_at = \$1, updated_at = CURRENT_TIMESTAMP WHERE id = \$2`

	t.Run("success", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectExec(updateQuery).
			WithArgs(at, int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		require.NoError(t, r.MarkEmailVerified(ctx, 5, at))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectExec(updateQuery).
			WithArgs(at, int64(5)).
			WillReturnError(errors.New("connection lost"))

		require.Error(t, r.MarkEmailVerified(ctx, 5, at))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
