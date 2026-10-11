package code

import (
	"context"
	"time"

	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Upsert(ctx context.Context, c *domain.EmailCode, cooldown time.Duration) error {
	return m.Called(ctx, c, cooldown).Error(0)
}

func (m *MockRepository) Attempt(ctx context.Context, userID int64, purpose string, now time.Time, maxAttempts int) (*domain.EmailCode, error) {
	args := m.Called(ctx, userID, purpose, now, maxAttempts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.EmailCode), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, userID int64, purpose string) error {
	return m.Called(ctx, userID, purpose).Error(0)
}
