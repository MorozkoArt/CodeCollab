package repository_test

import (
	"context"
	"testing"

	"github.com/MorozkoArt/CodeCollab/internal/config"
	"github.com/MorozkoArt/CodeCollab/internal/domain"
	"github.com/MorozkoArt/CodeCollab/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testEmail    = "test@example.com"
	testUsername = "testuser"
	testPassword = "password123"

	dupEmail    = "duplicate@example.com"
	dupUsername = "duplicate"

	foundEmail    = "found@example.com"
	foundUsername = "founduser"

	byIDEmail    = "byid@example.com"
	byIDUsername = "byiduser"

	notFoundEmail = "notfound@example.com"
	notFoundID    = int64(999999)
)

func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	cfg := config.NewDBConfig()

	pool, err := pgxpool.New(context.Background(), cfg.DSN())
	require.NoError(t, err, "failed to connect to database, check TEST_DB_HOST")
	require.NoError(t, pool.Ping(context.Background()))

	_, err = pool.Exec(context.Background(),
		"TRUNCATE TABLE users RESTART IDENTITY CASCADE",
	)
	require.NoError(t, err)

	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestUserRepository_Create(t *testing.T) {
	repo := repository.NewUserRepository(setupDB(t))
	ctx := context.Background()

	t.Run("creates user successfully", func(t *testing.T) {
		err := repo.Create(ctx, &domain.User{
			Email:    testEmail,
			Username: testUsername,
			Password: testPassword,
		})
		require.NoError(t, err)
	})

	t.Run("returns error when user exists", func(t *testing.T) {
		user := &domain.User{
			Email:    dupEmail,
			Username: dupUsername,
			Password: testPassword,
		}
		require.NoError(t, repo.Create(ctx, user))

		err := repo.Create(ctx, user)
		assert.ErrorIs(t, err, repository.ErrUserExists)
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	repo := repository.NewUserRepository(setupDB(t))
	ctx := context.Background()

	t.Run("returns user when found", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, &domain.User{
			Email:    foundEmail,
			Username: foundUsername,
			Password: testPassword,
		}))

		user, err := repo.GetByEmail(ctx, foundEmail)
		require.NoError(t, err)
		assert.Equal(t, foundEmail, user.Email)
		assert.Equal(t, foundUsername, user.Username)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		_, err := repo.GetByEmail(ctx, notFoundEmail)
		assert.ErrorIs(t, err, repository.ErrUserNotFound)
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	repo := repository.NewUserRepository(setupDB(t))
	ctx := context.Background()

	t.Run("returns user when found", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, &domain.User{
			Email:    byIDEmail,
			Username: byIDUsername,
			Password: testPassword,
		}))

		created, err := repo.GetByEmail(ctx, byIDEmail)
		require.NoError(t, err)

		user, err := repo.GetByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, user.ID)
		assert.Equal(t, byIDEmail, user.Email)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, notFoundID)
		assert.ErrorIs(t, err, repository.ErrUserNotFound)
	})
}
