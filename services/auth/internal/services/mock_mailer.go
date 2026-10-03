package services

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockMailer struct {
	mock.Mock
}

func (m *MockMailer) SendCode(ctx context.Context, to, purpose, code string, ttl time.Duration) error {
	args := m.Called(ctx, to, purpose, code, ttl)
	return args.Error(0)
}
