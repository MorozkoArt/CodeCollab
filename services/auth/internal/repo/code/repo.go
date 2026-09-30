package code

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	pkgdb "github.com/MorozkoArt/CodeCollab/pkg/db"
	"github.com/jackc/pgx/v4"
)

var (
	ErrCodeNotFound = errors.New("code not found or expired")
	ErrCooldown     = errors.New("code was sent recently")
)

type CodeRepository interface {
	Upsert(ctx context.Context, userID int64, purpose string, hash []byte, now time.Time, ttl, cooldown time.Duration) error
	Attempt(ctx context.Context, userID int64, purpose string, now time.Time, maxAttempts int) ([]byte, error)
	Delete(ctx context.Context, userID int64, purpose string) error
}

type codeRepository struct {
	db      pkgdb.SQL
	builder squirrel.StatementBuilderType
}

func NewCodeRepository(sqlDB pkgdb.SQL, builder squirrel.StatementBuilderType) CodeRepository {
	return &codeRepository{db: sqlDB, builder: builder.PlaceholderFormat(squirrel.Dollar)}
}

func (r *codeRepository) Upsert(ctx context.Context, userID int64, purpose string, hash []byte, now time.Time, ttl, cooldown time.Duration) error {
	query, args, err := r.builder.
		Insert("email_codes").
		Columns("user_id", "purpose", "code_hash", "expires_at", "created_at").
		Values(userID, purpose, hash, now.Add(ttl), now).
		Suffix(`ON CONFLICT (user_id, purpose) DO UPDATE SET
			code_hash = EXCLUDED.code_hash,
			expires_at = EXCLUDED.expires_at,
			attempts = 0,
			created_at = EXCLUDED.created_at
			WHERE email_codes.created_at <= ?`, now.Add(-cooldown)).
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

func (r *codeRepository) Attempt(ctx context.Context, userID int64, purpose string, now time.Time, maxAttempts int) ([]byte, error) {
	query, args, err := r.builder.
		Update("email_codes").
		Set("attempts", squirrel.Expr("attempts + 1")).
		Where(squirrel.Eq{"user_id": userID, "purpose": purpose}).
		Where(squirrel.Gt{"expires_at": now}).
		Where(squirrel.Lt{"attempts": maxAttempts}).
		Suffix("RETURNING code_hash").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build attempt sql: %w", err)
	}

	var hash []byte
	err = r.db.QueryRow(ctx, query, args...).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("code attempt: %w", err)
	}
	return hash, nil
}

func (r *codeRepository) Delete(ctx context.Context, userID int64, purpose string) error {
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
