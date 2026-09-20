package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	pkgdb "github.com/MorozkoArt/CodeCollab/pkg/db"
	"github.com/MorozkoArt/CodeCollab/pkg/password"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
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

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	query, args, err := r.builder.
		Select("COUNT(*)").
		From("users").
		Where(squirrel.Eq{"email": u.Email}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build exists sql: %w", err)
	}

	var count int
	if err = r.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		log.Error().Err(err).Ctx(ctx).Str("email", u.Email).Msg("Failed to check user existence")
		return err
	}
	if count > 0 {
		return ErrUserExists
	}

	hashed, err := password.Hash(u.Password)
	if err != nil {
		log.Error().Err(err).Ctx(ctx).Msg("Failed to hash password")
		return err
	}

	query, args, err = r.builder.
		Insert("users").
		Columns("username", "email", "password").
		Values(u.Username, u.Email, hashed).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert sql: %w", err)
	}

	if _, err = r.db.Exec(ctx, query, args...); err != nil {
		log.Error().Err(err).Ctx(ctx).Str("email", u.Email).Msg("Failed to create user")
		return err
	}

	log.Info().Ctx(ctx).Str("email", u.Email).Msg("User created successfully")
	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query, args, err := r.builder.
		Select("id", "username", "email", "password").
		From("users").
		Where(squirrel.Eq{"email": email}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select sql: %w", err)
	}

	var u domain.User
	err = r.db.QueryRow(ctx, query, args...).Scan(&u.ID, &u.Username, &u.Email, &u.Password)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		log.Error().Err(err).Ctx(ctx).Str("email", email).Msg("Failed to get user by email")
		return nil, err
	}

	return &u, nil
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	query, args, err := r.builder.
		Select("id", "username", "email").
		From("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select sql: %w", err)
	}

	var u domain.User
	err = r.db.QueryRow(ctx, query, args...).Scan(&u.ID, &u.Username, &u.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		log.Error().Err(err).Ctx(ctx).Int64("id", id).Msg("Failed to get user by id")
		return nil, err
	}

	return &u, nil
}
