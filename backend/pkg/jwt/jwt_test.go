package jwt_test

import (
	"testing"
	"time"

	"github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSecret    = "test_secret"
	testExpiry    = time.Hour
	expiredExpiry = -time.Hour
	testEmail     = "test@example.com"
	testUserID    = int64(1)
	invalidToken  = "invalid.token"
)

var testSvc = jwt.NewService(testSecret, testExpiry)

func TestGenerateAndValidateToken(t *testing.T) {
	token, err := testSvc.GenerateToken(testUserID, testEmail)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := testSvc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, testUserID, claims.UserID)
	assert.Equal(t, testEmail, claims.Email)
}

func TestValidateToken_Invalid(t *testing.T) {
	_, err := testSvc.ValidateToken(invalidToken)
	assert.Error(t, err)
}

func TestValidateToken_Expired(t *testing.T) {
	svc := jwt.NewService(testSecret, expiredExpiry)
	token, err := svc.GenerateToken(testUserID, testEmail)
	require.NoError(t, err)

	_, err = testSvc.ValidateToken(token)
	assert.Error(t, err)
}
