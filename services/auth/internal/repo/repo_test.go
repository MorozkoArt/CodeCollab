package repo_test

import (
	"context"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/repo"
	"github.com/jackc/pgx/v4"
	"github.com/pashagolub/pgxmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockRepo(t *testing.T) (repo.UserRepository, pgxmock.PgxConnIface) {
	t.Helper()

	mock, err := pgxmock.NewConn()
	require.NoError(t, err)

	t.Cleanup(func() { mock.Close(context.Background()) })

	r := repo.NewUserRepository(adapter{PgxConnIface: mock}, squirrel.StatementBuilder)
	return r, mock
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

	t.Run("creates user successfully", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectExec(`INSERT INTO users \(username,email,password\) VALUES \(\$1,\$2,\$3\)`).
			WithArgs("testuser", "test@example.com", "hashed_password").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := r.Create(ctx, &domain.User{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "hashed_password",
		})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("returns user when found", func(t *testing.T) {
		r, mock := newMockRepo(t)

		now := time.Now().Truncate(time.Second)
		mock.ExpectQuery(`SELECT id, username, email, password, created_at, updated_at FROM users WHERE email = \$1`).
			WithArgs("test@example.com").
			WillReturnRows(pgxmock.NewRows([]string{"id", "username", "email", "password", "created_at", "updated_at"}).
				AddRow(int64(1), "testuser", "test@example.com", "hashed", now, now))

		user, err := r.GetByEmail(ctx, "test@example.com")
		require.NoError(t, err)
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "test@example.com", user.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when not found", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(`SELECT id, username, email, password, created_at, updated_at FROM users WHERE email = \$1`).
			WithArgs("notfound@example.com").
			WillReturnError(pgx.ErrNoRows)

		_, err := r.GetByEmail(ctx, "notfound@example.com")
		assert.ErrorIs(t, err, repo.ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns user when found", func(t *testing.T) {
		r, mock := newMockRepo(t)

		now := time.Now().Truncate(time.Second)
		mock.ExpectQuery(`SELECT id, username, email, created_at, updated_at FROM users WHERE id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "username", "email", "created_at", "updated_at"}).
				AddRow(int64(1), "testuser", "test@example.com", now, now))

		user, err := r.GetByID(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when not found", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(`SELECT id, username, email, created_at, updated_at FROM users WHERE id = \$1`).
			WithArgs(int64(999999)).
			WillReturnError(pgx.ErrNoRows)

		_, err := r.GetByID(ctx, 999999)
		assert.ErrorIs(t, err, repo.ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
