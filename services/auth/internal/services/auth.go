package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	pkgjwt "github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/pkg/password"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/otp"
	codeRepo "github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/code"
	userRepo "github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/user"
	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/enum"
)

const (
	codeTTL         = 10 * time.Minute
	resendCooldown  = time.Minute
	maxCodeAttempts = 5
)

var (
	ErrUserExists       = errors.New("user already exists")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrEmailNotVerified = errors.New("email not verified")
	ErrInvalidCode      = errors.New("invalid or expired code")
	ErrTooManyRequests  = errors.New("too many requests")
)

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

type Mailer interface {
	SendCode(ctx context.Context, to, purpose, code string, ttl time.Duration) error
}

type AuthService interface {
	Register(ctx context.Context, input RegisterInput) error
	ConfirmRegistration(ctx context.Context, email, code string) error
	Login(ctx context.Context, email, pass string) error
	VerifyLogin(ctx context.Context, email, code string) (string, error)
	Validate(ctx context.Context, token string) (int64, error)
	GetUser(ctx context.Context, id int64) (*domain.User, error)
}

type authService struct {
	users     userRepo.UserRepository
	codes     codeRepo.CodeRepository
	jwtSvc    pkgjwt.Service
	mailer    Mailer
	otpSecret []byte
}

func NewAuthService(users userRepo.UserRepository, codes codeRepo.CodeRepository, jwtSvc pkgjwt.Service, mailer Mailer, otpSecret []byte) AuthService {
	return &authService{users: users, codes: codes, jwtSvc: jwtSvc, mailer: mailer, otpSecret: otpSecret}
}

func (s *authService) Register(ctx context.Context, in RegisterInput) error {
	u, err := s.users.GetByEmail(ctx, in.Email)
	switch {
	case err == nil && u.EmailVerifiedAt != nil:
		return ErrUserExists
	case err == nil:
		// Email заведён, но не подтверждён. Данные не трогаем: иначе можно подменить
		// пароль чужой незавершённой регистрации. Только шлём код заново.
	case errors.Is(err, userRepo.ErrUserNotFound):
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

	return s.issueCode(ctx, u, enum.PurposeRegister)
}

func (s *authService) ConfirmRegistration(ctx context.Context, email, code string) error {
	u, err := s.userForCode(ctx, email)
	if err != nil {
		return err
	}
	if err = s.verifyCode(ctx, u, enum.PurposeRegister, code); err != nil {
		return err
	}
	if err = s.users.MarkEmailVerified(ctx, u.ID, time.Now()); err != nil {
		return fmt.Errorf("mark verified: %w", err)
	}
	return nil
}

func (s *authService) Login(ctx context.Context, email, pass string) error {
	u, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, userRepo.ErrUserNotFound) {
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
	return s.issueCode(ctx, u, enum.PurposeLogin)
}

func (s *authService) VerifyLogin(ctx context.Context, email, code string) (string, error) {
	u, err := s.userForCode(ctx, email)
	if err != nil {
		return "", err
	}
	if err = s.verifyCode(ctx, u, enum.PurposeLogin, code); err != nil {
		return "", err
	}
	return s.jwtSvc.GenerateToken(u.ID, u.Email)
}

func (s *authService) Validate(ctx context.Context, token string) (int64, error) {
	claims, err := s.jwtSvc.ValidateToken(token)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

func (s *authService) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (s *authService) userForCode(ctx context.Context, email string) (*domain.User, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, userRepo.ErrUserNotFound) {
		return nil, ErrInvalidCode
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (s *authService) issueCode(ctx context.Context, u *domain.User, purpose string) error {
	code, err := otp.Generate()
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	err = s.codes.Upsert(ctx, u.ID, purpose, otp.Hash(s.otpSecret, u.ID, purpose, code), time.Now(), codeTTL, resendCooldown)
	if errors.Is(err, codeRepo.ErrCooldown) {
		return ErrTooManyRequests
	}
	if err != nil {
		return fmt.Errorf("store code: %w", err)
	}

	if err = s.mailer.SendCode(ctx, u.Email, purpose, code, codeTTL); err != nil {
		_ = s.codes.Delete(ctx, u.ID, purpose)
		return fmt.Errorf("send code: %w", err)
	}
	return nil
}

func (s *authService) verifyCode(ctx context.Context, u *domain.User, purpose, code string) error {
	stored, err := s.codes.Attempt(ctx, u.ID, purpose, time.Now(), maxCodeAttempts)
	if errors.Is(err, codeRepo.ErrCodeNotFound) {
		return ErrInvalidCode
	}
	if err != nil {
		return fmt.Errorf("check code: %w", err)
	}

	if !otp.Equal(stored, otp.Hash(s.otpSecret, u.ID, purpose, code)) {
		return ErrInvalidCode
	}

	if err = s.codes.Delete(ctx, u.ID, purpose); err != nil {
		return fmt.Errorf("consume code: %w", err)
	}
	return nil
}

func mapUserErr(op string, err error) error {
	if errors.Is(err, userRepo.ErrUserExists) {
		return ErrUserExists
	}
	return fmt.Errorf("%s: %w", op, err)
}
