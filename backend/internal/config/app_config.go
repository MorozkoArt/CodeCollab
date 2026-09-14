package config

import "github.com/MorozkoArt/CodeCollab/pkg/env"

type AppConfig struct {
	AppEnv string
}

func NewAppConfig() *AppConfig {
	return &AppConfig{
		AppEnv: env.Get("APP_ENV", defaultAppEnv),
	}
}
