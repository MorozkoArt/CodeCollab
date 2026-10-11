package config

import "github.com/MorozkoArt/CodeCollab/pkg/env"

type App struct {
	env  string
	name string
}

func NewAppConfig() *App {
	return &App{
		env:  env.GetEnv("APP_ENV", defaultAppEnv),
		name: env.GetEnv("APP_NAME", defaultAppName),
	}
}

func (c *App) Env() string  { return c.env }
func (c *App) Name() string { return c.name }
