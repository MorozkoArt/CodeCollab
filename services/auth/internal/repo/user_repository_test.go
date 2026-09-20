package repo_test

import (
	"context"
	"testing"

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

func TestUserRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates user successfully", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE email = \$1`).
			WithArgs("test@example.com").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectExec(`INSERT INTO users \(username,email,password\) VALUES \(\$1,\$2,\$3\)`).
			WithArgs("testuser", "test@example.com", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := r.Create(ctx, &domain.User{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "password123",
		})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when user exists", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE email = \$1`).
			WithArgs("test@example.com").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

		err := r.Create(ctx, &domain.User{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "password123",
		})
		assert.ErrorIs(t, err, repo.ErrUserExists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("returns user when found", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(`SELECT id, username, email, password FROM users WHERE email = \$1`).
			WithArgs("test@example.com").
			WillReturnRows(pgxmock.NewRows([]string{"id", "username", "email", "password"}).
				AddRow(int64(1), "testuser", "test@example.com", "hashed"))

		user, err := r.GetByEmail(ctx, "test@example.com")
		require.NoError(t, err)
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "test@example.com", user.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when not found", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(`SELECT id, username, email, password FROM users WHERE email = \$1`).
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

		mock.ExpectQuery(`SELECT id, username, email FROM users WHERE id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "username", "email"}).
				AddRow(int64(1), "testuser", "test@example.com"))

		user, err := r.GetByID(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when not found", func(t *testing.T) {
		r, mock := newMockRepo(t)

		mock.ExpectQuery(`SELECT id, username, email FROM users WHERE id = \$1`).
			WithArgs(int64(999999)).
			WillReturnError(pgx.ErrNoRows)

		_, err := r.GetByID(ctx, 999999)
		assert.ErrorIs(t, err, repo.ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
