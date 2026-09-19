package config

import "github.com/MorozkoArt/CodeCollab/pkg/env"

type ServerConfig struct {
	port int
	host string
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		port: env.GetInt("SERVER_PORT", defaultServerPort),
		host: env.Get("SERVER_HOST", defaultServerHost),
	}
}

func (c *ServerConfig) Port() int    { return c.port }
func (c *ServerConfig) Host() string { return c.host }
