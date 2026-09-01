package config

import "github.com/MorozkoArt/CodeCollab/pkg/env"

type Config struct {
	*ServerConfig
	*DBConfig
	*AuthConfig
	AppEnv string
}

func NewConfig() *Config {
	return &Config{
		ServerConfig: NewServerConfig(),
		DBConfig:     NewDBConfig(),
		AuthConfig:   NewAuthConfig(),
		AppEnv:       env.Get("APP_ENV", "development"),
	}
}
