package services

import (
	"context"
	"errors"

	"github.com/MorozkoArt/CodeCollab/internal/domain"
	"github.com/MorozkoArt/CodeCollab/internal/repository"
	"github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo   repository.UserRepository
	jwtService *jwt.Service
}

func NewAuthService(userRepo repository.UserRepository, jwtService *jwt.Service) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

func (s *AuthService) Register(ctx context.Context, req *domain.RegisterRequest) error {
	log.Info().Ctx(ctx).Str("email", req.Email).Msg("Registering user")

	user := &domain.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	return s.userRepo.Create(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.UserResponse, string, error) {
	log.Info().Ctx(ctx).Str("email", req.Email).Msg("Login attempt")

	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		log.Warn().Ctx(ctx).Str("email", req.Email).Msg("Login failed: user not found")
		return nil, "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		log.Warn().Ctx(ctx).Str("email", req.Email).Msg("Login failed: invalid password")
		return nil, "", errors.New("invalid credentials")
	}

	token, err := s.jwtService.GenerateToken(user.ID, user.Email)
	if err != nil {
		log.Error().Err(err).Ctx(ctx).Msg("Failed to generate token")
		return nil, "", err
	}

	log.Info().Ctx(ctx).Str("email", req.Email).Msg("Login successful")

	return &domain.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	}, token, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*jwt.Claims, error) {
	return s.jwtService.ValidateToken(tokenString)
}
