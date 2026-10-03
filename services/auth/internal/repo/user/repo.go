package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	pkgdb "github.com/MorozkoArt/CodeCollab/pkg/db"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type UserRepository interface {
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	MarkEmailVerified(ctx context.Context, id int64, at time.Time) error
}

type userRepository struct {
	db      pkgdb.SQL
	builder squirrel.StatementBuilderType
}

func NewUserRepository(sqlDB pkgdb.SQL, builder squirrel.StatementBuilderType) UserRepository {
	return &userRepository{
		db:      sqlDB,
		builder: builder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query, args, err := r.builder.
		Select("COUNT(1)").
		From("users").
		Where(squirrel.Eq{"email": email}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build exists sql: %w", err)
	}

	var count int
	if err = r.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("exists by email: %w", err)
	}

	return count > 0, nil
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	query, args, err := r.builder.
		Insert("users").
		Columns("username", "email", "password").
		Values(u.Username, u.Email, u.Password).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert sql: %w", err)
	}

	if err = r.db.QueryRow(ctx, query, args...).Scan(&u.ID); err != nil {
		return mapWriteErr("create user", err)
	}
	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query, args, err := r.builder.
		Select("id", "username", "email", "password", "email_verified_at", "created_at", "updated_at").
		From("users").
		Where(squirrel.Eq{"email": email}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select sql: %w", err)
	}

	var u domain.User
	err = r.db.QueryRow(ctx, query, args...).
		Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.EmailVerifiedAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &u, nil
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	query, args, err := r.builder.
		Select("id", "username", "email", "created_at", "updated_at").
		From("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select sql: %w", err)
	}

	var u domain.User
	err = r.db.QueryRow(ctx, query, args...).
		Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &u, nil
}

func (r *userRepository) MarkEmailVerified(ctx context.Context, id int64, at time.Time) error {
	return r.exec(ctx, "mark email verified", r.builder.
		Update("users").
		SetMap(map[string]any{"email_verified_at": at, "updated_at": squirrel.Expr("CURRENT_TIMESTAMP")}).
		Where(squirrel.Eq{"id": id}))
}

func (r *userRepository) exec(ctx context.Context, op string, b squirrel.UpdateBuilder) error {
	query, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("%s: build sql: %w", op, err)
	}
	if _, err = r.db.Exec(ctx, query, args...); err != nil {
		return mapWriteErr(op, err)
	}
	return nil
}

func mapWriteErr(op string, err error) error {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return ErrUserExists
	}
	return fmt.Errorf("%s: %w", op, err)
}
