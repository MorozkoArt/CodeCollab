package config

import "github.com/MorozkoArt/CodeCollab/pkg/env"

type App struct {
	AppEnv string
}

func NewAppConfig() *App {
	return &App{
		AppEnv: env.Get("APP_ENV", defaultAppEnv),
	}
}
