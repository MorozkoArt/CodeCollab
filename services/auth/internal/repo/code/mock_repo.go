package code

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockCodeRepository struct {
	mock.Mock
}

func (m *MockCodeRepository) Upsert(ctx context.Context, userID int64, purpose string, hash []byte, now time.Time, ttl, cooldown time.Duration) error {
	args := m.Called(ctx, userID, purpose, hash, now, ttl, cooldown)
	return args.Error(0)
}

func (m *MockCodeRepository) Attempt(ctx context.Context, userID int64, purpose string, now time.Time, maxAttempts int) ([]byte, error) {
	args := m.Called(ctx, userID, purpose, now, maxAttempts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCodeRepository) Delete(ctx context.Context, userID int64, purpose string) error {
	args := m.Called(ctx, userID, purpose)
	return args.Error(0)
}
