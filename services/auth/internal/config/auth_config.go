package config

import (
	"time"

	"github.com/MorozkoArt/CodeCollab/pkg/env"
)

type AuthConfig struct {
	jwtSecret   string
	tokenExpiry time.Duration
}

func NewAuthConfig() *AuthConfig {
	return &AuthConfig{
		jwtSecret:   env.Get("JWT_SECRET", ""),
		tokenExpiry: env.GetDuration("JWT_EXPIRY", defaultTokenExpiry),
	}
}

func (c *AuthConfig) JWTSecret() string          { return c.jwtSecret }
func (c *AuthConfig) TokenExpiry() time.Duration { return c.tokenExpiry }
