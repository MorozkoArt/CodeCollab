package repository

import (
	"context"
	"errors"

	"github.com/MorozkoArt/CodeCollab/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
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
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", u.Email,
	).Scan(&exists)
	if err != nil {
		log.Error().Err(err).Ctx(ctx).Str("email", u.Email).Msg("Failed to check user existence")
		return err
	}
	if exists {
		return ErrUserExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Ctx(ctx).Msg("Failed to hash password")
		return err
	}

	_, err = r.db.Exec(ctx,
		"INSERT INTO users (username, email, password) VALUES ($1, $2, $3)",
		u.Username, u.Email, string(hashed),
	)
	if err != nil {
		log.Error().Err(err).Ctx(ctx).Str("email", u.Email).Msg("Failed to create user")
		return err
	}

	log.Info().Ctx(ctx).Str("email", u.Email).Msg("User created successfully")
	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		"SELECT id, username, email, password FROM users WHERE email = $1", email,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password)

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
	var u domain.User
	err := r.db.QueryRow(ctx,
		"SELECT id, username, email FROM users WHERE id = $1", id,
	).Scan(&u.ID, &u.Username, &u.Email)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		log.Error().Err(err).Ctx(ctx).Int64("id", id).Msg("Failed to get user by id")
		return nil, err
	}

	return &u, nil
}
