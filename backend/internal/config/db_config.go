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
	return &DBConfig{
		host:     env.Get("DB_HOST", "localhost"),
		port:     env.GetInt("DB_PORT", 5432),
		user:     env.Get("DB_USER", "app_user"),
		password: env.Get("DB_PASSWORD", ""),
		name:     env.Get("DB_NAME", "codecollab"),
		sslMode:  env.Get("DB_SSLMODE", "disable"),
	}
}

func (c *DBConfig) DSN() string {
	hostPort := net.JoinHostPort(c.host, strconv.Itoa(c.port))

	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		c.user,
		c.password,
		hostPort,
		c.name,
		c.sslMode,
	)
}
