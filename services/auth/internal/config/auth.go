package config

import (
	"time"

	"github.com/MorozkoArt/CodeCollab/pkg/env"
)

type Auth struct {
	jwtSecret   string
	tokenExpiry time.Duration
}

func NewAuthConfig() *Auth {
	return &Auth{
		jwtSecret:   env.GetEnv("JWT_SECRET", ""),
		tokenExpiry: env.GetDurationEnv("JWT_EXPIRY", defaultTokenExpiry),
	}
}

func (c *Auth) JWTSecret() string          { return c.jwtSecret }
func (c *Auth) TokenExpiry() time.Duration { return c.tokenExpiry }
