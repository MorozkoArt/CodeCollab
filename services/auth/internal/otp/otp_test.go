package otp_test

import (
	"regexp"
	"testing"

	"github.com/MorozkoArt/CodeCollab/services/auth/internal/otp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	re := regexp.MustCompile(`^\d{6}$`)
	seen := map[string]struct{}{}
	for range 200 {
		code, err := otp.Generate()
		require.NoError(t, err)
		assert.Regexp(t, re, code)
		seen[code] = struct{}{}
	}
	assert.Greater(t, len(seen), 150, "codes should be random")
}

func TestHash(t *testing.T) {
	secret := []byte("secret-secret-secret-secret-1234")
	base := otp.Hash(secret, 1, "login", "123456")

	assert.True(t, otp.Equal(base, otp.Hash(secret, 1, "login", "123456")), "deterministic")
	assert.False(t, otp.Equal(base, otp.Hash(secret, 2, "login", "123456")), "bound to user")
	assert.False(t, otp.Equal(base, otp.Hash(secret, 1, "register", "123456")), "bound to purpose")
	assert.False(t, otp.Equal(base, otp.Hash(secret, 1, "login", "654321")), "depends on code")
	assert.False(t, otp.Equal(base, otp.Hash([]byte("other-secret-other-secret-other-1"), 1, "login", "123456")), "depends on secret")
}
