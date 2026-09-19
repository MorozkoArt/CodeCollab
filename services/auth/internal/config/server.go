package config

import "github.com/MorozkoArt/CodeCollab/pkg/env"

type Server struct {
	port     int
	host     string
	grpcPort int
}

func NewServerConfig() *Server {
	return &Server{
		port:     env.GetIntEnv("SERVER_PORT", defaultServerPort),
		host:     env.GetEnv("SERVER_HOST", defaultServerHost),
		grpcPort: env.GetIntEnv("GRPC_PORT", defaultGRPCPort),
	}
}

func (c *Server) Port() int     { return c.port }
func (c *Server) Host() string  { return c.host }
func (c *Server) GRPCPort() int { return c.grpcPort }
