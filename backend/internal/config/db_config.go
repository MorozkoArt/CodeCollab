package config

import (
	"fmt"
	"net"
	"strconv"

	"github.com/MorozkoArt/CodeCollab/pkg/env"
)

type DBConfig struct {
	host     string
	port     int
	user     string
	password string
	name     string
	sslMode  string
}

func NewDBConfig() *DBConfig {
	host := env.Get("TEST_DB_HOST", "")
	if host == "" {
		host = env.Get("DB_HOST", defaultDBHost)
	}

	return &DBConfig{
		host:     host,
		port:     env.GetInt("POSTGRES_PORT", defaultDBPort),
		user:     env.Get("POSTGRES_USER", defaultDBUser),
		password: env.Get("POSTGRES_PASSWORD", ""),
		name:     env.Get("POSTGRES_DB", defaultDBName),
		sslMode:  env.Get("DB_SSLMODE", defaultDBSSLMode),
	}
}

func (c *DBConfig) DSN() string {
	hostPort := net.JoinHostPort(c.host, strconv.Itoa(c.port))
	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		c.user, c.password, hostPort, c.name, c.sslMode,
	)
}
