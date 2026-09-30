package code

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/otp"
	coderepo "github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/code"
)

const (
	codeTTL         = 10 * time.Minute
	resendCooldown  = time.Minute
	maxCodeAttempts = 5
)

var (
	ErrInvalidCode     = errors.New("invalid or expired code")
	ErrTooManyRequests = errors.New("too many requests")
)

type Mailer interface {
	SendCode(ctx context.Context, to, purpose, code string, ttl time.Duration) error
}

type Service interface {
	Issue(ctx context.Context, u *domain.User, purpose string) error
	Verify(ctx context.Context, u *domain.User, purpose, code string) error
}

type service struct {
	codes  coderepo.Repository
	mailer Mailer
	secret []byte
}

func NewService(codes coderepo.Repository, mailer Mailer, secret []byte) Service {
	return &service{codes: codes, mailer: mailer, secret: secret}
}

func (s *service) Issue(ctx context.Context, u *domain.User, purpose string) error {
	code, err := otp.Generate()
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	now := time.Now()
	ec := &domain.EmailCode{
		UserID:    u.ID,
		Purpose:   purpose,
		CodeHash:  otp.Hash(s.secret, u.ID, purpose, code),
		ExpiresAt: now.Add(codeTTL),
		CreatedAt: now,
	}

	err = s.codes.Upsert(ctx, ec, resendCooldown)
	if errors.Is(err, coderepo.ErrCooldown) {
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

func (s *service) Verify(ctx context.Context, u *domain.User, purpose, code string) error {
	stored, err := s.codes.Attempt(ctx, u.ID, purpose, time.Now(), maxCodeAttempts)
	if errors.Is(err, coderepo.ErrCodeNotFound) {
		return ErrInvalidCode
	}
	if err != nil {
		return fmt.Errorf("check code: %w", err)
	}

	if !otp.Equal(stored.CodeHash, otp.Hash(s.secret, u.ID, purpose, code)) {
		return ErrInvalidCode
	}

	if err = s.codes.Delete(ctx, u.ID, purpose); err != nil {
		return fmt.Errorf("consume code: %w", err)
	}
	return nil
}
