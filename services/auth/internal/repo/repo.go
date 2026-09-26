package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	pkgdb "github.com/MorozkoArt/CodeCollab/pkg/db"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	dbmodel "github.com/MorozkoArt/CodeCollab/services/auth/internal/models/db"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	ExistsByEmail(ctx context.Context, email string) (bool, error)
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
		log.Error().Err(err).Ctx(ctx).Str("email", email).Msg("Failed to check user existence")
		return false, err
	}

	return count > 0, nil
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	query, args, err := r.builder.
		Insert("users").
		Columns("username", "email", "password").
		Values(u.Username, u.Email, u.Password).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert sql: %w", err)
	}

	if _, err = r.db.Exec(ctx, query, args...); err != nil {
		log.Error().Err(err).Ctx(ctx).Str("email", u.Email).Msg("Failed to create user")
		return err
	}

	log.Info().Ctx(ctx).Str("email", u.Email).Msg("User created")
	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query, args, err := r.builder.
		Select("id", "username", "email", "password", "created_at", "updated_at").
		From("users").
		Where(squirrel.Eq{"email": email}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select sql: %w", err)
	}

	var u dbmodel.User
	err = r.db.QueryRow(ctx, query, args...).
		Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		log.Error().Err(err).Ctx(ctx).Str("email", email).Msg("Failed to get user by email")
		return nil, err
	}

	return toDomain(&u), nil
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

	var u dbmodel.User
	err = r.db.QueryRow(ctx, query, args...).
		Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		log.Error().Err(err).Ctx(ctx).Int64("id", id).Msg("Failed to get user by id")
		return nil, err
	}

	return toDomain(&u), nil
}

func toDomain(u *dbmodel.User) *domain.User {
	return &domain.User{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Password:  u.Password,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
