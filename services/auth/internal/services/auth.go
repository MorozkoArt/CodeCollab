package services

import (
	"context"
	"errors"
	"fmt"

	pkgjwt "github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/pkg/password"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/repo"
)

var (
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid password")
)

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

type AuthService interface {
	Register(ctx context.Context, input RegisterInput) error
	Login(ctx context.Context, email, pass string) (string, error)
	Validate(ctx context.Context, token string) (int64, error)
	GetUser(ctx context.Context, id int64) (*domain.User, error)
}

type authService struct {
	repo   repo.UserRepository
	jwtSvc pkgjwt.Service
}

func NewAuthService(userRepo repo.UserRepository, jwtSvc pkgjwt.Service) AuthService {
	return &authService{repo: userRepo, jwtSvc: jwtSvc}
}

func (s *authService) Register(ctx context.Context, input RegisterInput) error {
	exists, err := s.repo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return fmt.Errorf("check user existence: %w", err)
	}
	if exists {
		return ErrUserExists
	}

	hashed, err := password.Hash(input.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.repo.Create(ctx, &domain.User{
		Username: input.Username,
		Email:    input.Email,
		Password: hashed,
	}); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (s *authService) Login(ctx context.Context, email, pass string) (string, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return "", ErrInvalidPassword
		}
		return "", fmt.Errorf("get user: %w", err)
	}

	if !password.Check(pass, user.Password) {
		return "", ErrInvalidPassword
	}

	return s.jwtSvc.GenerateToken(user.ID, user.Email)
}

func (s *authService) Validate(ctx context.Context, token string) (int64, error) {
	claims, err := s.jwtSvc.ValidateToken(token)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

func (s *authService) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}
