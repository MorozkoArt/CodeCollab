package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	pkgjwt "github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/pkg/password"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	userrepo "github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/user"
	codesvc "github.com/MorozkoArt/CodeCollab/services/auth/internal/services/code"
	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/enum"
)

var (
	ErrUserExists       = errors.New("user already exists")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrEmailNotVerified = errors.New("email not verified")
)

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

type Service interface {
	Register(ctx context.Context, input RegisterInput) error
	ConfirmRegistration(ctx context.Context, email, code string) error
	Login(ctx context.Context, email, pass string) error
	VerifyLogin(ctx context.Context, email, code string) (string, error)
	Validate(ctx context.Context, token string) (int64, error)
	GetUser(ctx context.Context, id int64) (*domain.User, error)
}

type service struct {
	users  userrepo.Repository
	codes  codesvc.Service
	jwtSvc pkgjwt.Service
}

func NewService(users userrepo.Repository, codes codesvc.Service, jwtSvc pkgjwt.Service) Service {
	return &service{users: users, codes: codes, jwtSvc: jwtSvc}
}

func (s *service) Register(ctx context.Context, in RegisterInput) error {
	u, err := s.users.GetByEmail(ctx, in.Email)
	switch {
	case err == nil && u.EmailVerifiedAt != nil:
		return ErrUserExists
	case err == nil:
		// Email заведён, но не подтверждён: данные не трогаем, только шлём код заново.
	case errors.Is(err, userrepo.ErrUserNotFound):
		hashed, hErr := password.Hash(in.Password)
		if hErr != nil {
			return fmt.Errorf("hash password: %w", hErr)
		}
		u = &domain.User{Username: in.Username, Email: in.Email, Password: hashed}
		if err = s.users.Create(ctx, u); err != nil {
			return mapUserErr("create user", err)
		}
	default:
		return fmt.Errorf("get user: %w", err)
	}

	return s.codes.Issue(ctx, u, enum.PurposeRegister)
}

func (s *service) ConfirmRegistration(ctx context.Context, email, code string) error {
	u, err := s.userForCode(ctx, email)
	if err != nil {
		return err
	}
	if err = s.codes.Verify(ctx, u, enum.PurposeRegister, code); err != nil {
		return err
	}
	if err = s.users.MarkEmailVerified(ctx, u.ID, time.Now()); err != nil {
		return fmt.Errorf("mark verified: %w", err)
	}
	return nil
}

func (s *service) Login(ctx context.Context, email, pass string) error {
	u, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, userrepo.ErrUserNotFound) {
		password.Dummy(pass)
		return ErrInvalidPassword
	}
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if !password.Check(pass, u.Password) {
		return ErrInvalidPassword
	}
	if u.EmailVerifiedAt == nil {
		return ErrEmailNotVerified
	}
	return s.codes.Issue(ctx, u, enum.PurposeLogin)
}

func (s *service) VerifyLogin(ctx context.Context, email, code string) (string, error) {
	u, err := s.userForCode(ctx, email)
	if err != nil {
		return "", err
	}
	if err = s.codes.Verify(ctx, u, enum.PurposeLogin, code); err != nil {
		return "", err
	}
	return s.jwtSvc.GenerateToken(u.ID, u.Email)
}

func (s *service) Validate(_ context.Context, token string) (int64, error) {
	claims, err := s.jwtSvc.ValidateToken(token)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

func (s *service) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (s *service) userForCode(ctx context.Context, email string) (*domain.User, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, userrepo.ErrUserNotFound) {
		return nil, codesvc.ErrInvalidCode
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func mapUserErr(op string, err error) error {
	if errors.Is(err, userrepo.ErrUserExists) {
		return ErrUserExists
	}
	return fmt.Errorf("%s: %w", op, err)
}
