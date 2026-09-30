package config

import (
	"time"

	"github.com/MorozkoArt/CodeCollab/pkg/env"
)

type Auth struct {
	jwtSecret   string
	tokenExpiry time.Duration
	otpSecret   string
}

func NewAuthConfig() *Auth {
	return &Auth{
		jwtSecret:   env.GetEnv("JWT_SECRET", ""),
		tokenExpiry: env.GetDurationEnv("JWT_EXPIRY", defaultTokenExpiry),
		otpSecret:   env.GetEnv("OTP_SECRET", ""),
	}
}

func (c *Auth) JWTSecret() string          { return c.jwtSecret }
func (c *Auth) TokenExpiry() time.Duration { return c.tokenExpiry }
func (c *Auth) OTPSecret() string          { return c.otpSecret }
