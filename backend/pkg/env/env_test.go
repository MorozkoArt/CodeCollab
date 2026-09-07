package env_test

import (
	"testing"
	"time"

	"github.com/MorozkoArt/CodeCollab/pkg/env"
	"github.com/stretchr/testify/assert"
)

const (
	keyTest            = "TEST_KEY"
	keyNotExisting     = "NOT_EXISTING_KEY"
	keyEmpty           = "EMPTY_KEY"
	keyTestInt         = "TEST_INT"
	keyNotExistingInt  = "NOT_EXISTING_INT"
	keyInvalidInt      = "INVALID_INT"
	keyTestDuration    = "TEST_DURATION"
	keyNotExistingDur  = "NOT_EXISTING_DURATION"
	keyInvalidDuration = "INVALID_DURATION"

	testValue       = "test_value"
	defaultValue    = "default"
	invalidValue    = "not_a_number"
	invalidDuration = "not_a_duration"

	testInt        = 42
	defaultInt     = 0
	defaultPort    = 8080
	defaultDBPort  = 5432
	testDuration   = 24 * time.Hour
	defaultTimeout = time.Hour
)

func TestGet(t *testing.T) {
	t.Run("returns env value when set", func(t *testing.T) {
		t.Setenv(keyTest, testValue)
		assert.Equal(t, testValue, env.Get(keyTest, defaultValue))
	})

	t.Run("returns default when not set", func(t *testing.T) {
		assert.Equal(t, defaultValue, env.Get(keyNotExisting, defaultValue))
	})

	t.Run("returns default when empty", func(t *testing.T) {
		t.Setenv(keyEmpty, "")
		assert.Equal(t, defaultValue, env.Get(keyEmpty, defaultValue))
	})
}

func TestGetInt(t *testing.T) {
	t.Run("returns int value when set", func(t *testing.T) {
		t.Setenv(keyTestInt, "42")
		assert.Equal(t, testInt, env.GetInt(keyTestInt, defaultInt))
	})

	t.Run("returns default when not set", func(t *testing.T) {
		assert.Equal(t, defaultPort, env.GetInt(keyNotExistingInt, defaultPort))
	})

	t.Run("returns default when invalid int", func(t *testing.T) {
		t.Setenv(keyInvalidInt, invalidValue)
		assert.Equal(t, defaultDBPort, env.GetInt(keyInvalidInt, defaultDBPort))
	})
}

func TestGetDuration(t *testing.T) {
	t.Run("returns duration when set", func(t *testing.T) {
		t.Setenv(keyTestDuration, "24h")
		assert.Equal(t, testDuration, env.GetDuration(keyTestDuration, defaultTimeout))
	})

	t.Run("returns default when not set", func(t *testing.T) {
		assert.Equal(t, defaultTimeout, env.GetDuration(keyNotExistingDur, defaultTimeout))
	})

	t.Run("returns default when invalid duration", func(t *testing.T) {
		t.Setenv(keyInvalidDuration, invalidDuration)
		assert.Equal(t, defaultTimeout, env.GetDuration(keyInvalidDuration, defaultTimeout))
	})
}
