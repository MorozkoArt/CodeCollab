package config

import "github.com/MorozkoArt/CodeCollab/pkg/env"

type App struct {
	env string
}

func NewAppConfig() *App {
	return &App{
		env: env.GetEnv("APP_ENV", defaultAppEnv),
	}
}

func (c *App) Env() string { return c.env }
