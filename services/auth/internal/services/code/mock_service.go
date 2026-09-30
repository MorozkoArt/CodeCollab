package code

import (
	"context"

	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockService struct{ mock.Mock }

func (m *MockService) Issue(ctx context.Context, u *domain.User, purpose string) error {
	return m.Called(ctx, u, purpose).Error(0)
}

func (m *MockService) Verify(ctx context.Context, u *domain.User, purpose, code string) error {
	return m.Called(ctx, u, purpose, code).Error(0)
}
