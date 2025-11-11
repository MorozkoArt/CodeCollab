package config

import "github.com/MorozkoArt/CodeCollab/pkg/env"

type ServerConfig struct {
	port int
	host string
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		port: env.GetInt("SERVER_PORT", 8080),
		host: env.Get("SERVER_HOST", "0.0.0.0"),
	}
}

func (c *ServerConfig) Port() int    { return c.port }
func (c *ServerConfig) Host() string { return c.host }
