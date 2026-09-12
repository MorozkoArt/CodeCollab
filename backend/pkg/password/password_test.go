package password_test

import (
	"testing"

	"github.com/MorozkoArt/CodeCollab/pkg/password"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testPassword  = "secret123"
	wrongPassword = "wrongpassword"
)

func TestHashAndCheck(t *testing.T) {
	hash, err := password.Hash(testPassword)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, password.Check(testPassword, hash))
}

func TestCheck_WrongPassword(t *testing.T) {
	hash, err := password.Hash(testPassword)
	require.NoError(t, err)
	assert.False(t, password.Check(wrongPassword, hash))
}
