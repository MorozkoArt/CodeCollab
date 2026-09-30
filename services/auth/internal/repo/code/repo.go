package code

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	pkgdb "github.com/MorozkoArt/CodeCollab/pkg/db"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/jackc/pgx/v4"
)

var (
	ErrCodeNotFound = errors.New("code not found or expired")
	ErrCooldown     = errors.New("code was sent recently")
)

type Repository interface {
	Upsert(ctx context.Context, c *domain.EmailCode, cooldown time.Duration) error
	Attempt(ctx context.Context, userID int64, purpose string, now time.Time, maxAttempts int) (*domain.EmailCode, error)
	Delete(ctx context.Context, userID int64, purpose string) error
}

type repository struct {
	db      pkgdb.SQL
	builder squirrel.StatementBuilderType
}

func NewRepository(sqlDB pkgdb.SQL, builder squirrel.StatementBuilderType) Repository {
	return &repository{db: sqlDB, builder: builder.PlaceholderFormat(squirrel.Dollar)}
}

func (r *repository) Upsert(ctx context.Context, c *domain.EmailCode, cooldown time.Duration) error {
	query, args, err := r.builder.
		Insert("email_codes").
		Columns("user_id", "purpose", "code_hash", "expires_at", "created_at").
		Values(c.UserID, c.Purpose, c.CodeHash, c.ExpiresAt, c.CreatedAt).
		Suffix(`ON CONFLICT (user_id, purpose) DO UPDATE SET
			code_hash = EXCLUDED.code_hash,
			expires_at = EXCLUDED.expires_at,
			attempts = 0,
			created_at = EXCLUDED.created_at
			WHERE email_codes.created_at <= ?`, c.CreatedAt.Add(-cooldown)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert code sql: %w", err)
	}

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("upsert code: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCooldown
	}
	return nil
}

func (r *repository) Attempt(ctx context.Context, userID int64, purpose string, now time.Time, maxAttempts int) (*domain.EmailCode, error) {
	query, args, err := r.builder.
		Update("email_codes").
		Set("attempts", squirrel.Expr("attempts + 1")).
		Where(squirrel.Eq{"user_id": userID, "purpose": purpose}).
		Where(squirrel.Gt{"expires_at": now}).
		Where(squirrel.Lt{"attempts": maxAttempts}).
		Suffix("RETURNING user_id, purpose, code_hash, attempts, expires_at, created_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build attempt sql: %w", err)
	}

	var c domain.EmailCode
	err = r.db.QueryRow(ctx, query, args...).
		Scan(&c.UserID, &c.Purpose, &c.CodeHash, &c.Attempts, &c.ExpiresAt, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("code attempt: %w", err)
	}
	return &c, nil
}

func (r *repository) Delete(ctx context.Context, userID int64, purpose string) error {
	query, args, err := r.builder.
		Delete("email_codes").
		Where(squirrel.Eq{"user_id": userID, "purpose": purpose}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete code sql: %w", err)
	}
	if _, err = r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete code: %w", err)
	}
	return nil
}
