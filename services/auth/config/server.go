package config

import "github.com/MorozkoArt/CodeCollab/pkg/env"

type Server struct {
	port int
	host string
}

func NewServerConfig() *Server {
	return &Server{
		port: env.GetInt("SERVER_PORT", defaultServerPort),
		host: env.Get("SERVER_HOST", defaultServerHost),
	}
}

func (c *Server) Port() int    { return c.port }
func (c *Server) Host() string { return c.host }
