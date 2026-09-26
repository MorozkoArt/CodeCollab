package config

import "github.com/MorozkoArt/CodeCollab/pkg/env"

type DB struct {
	host    string
	port    int
	user    string
	pass    string
	name    string
	sslMode string
}

func NewDBConfig() *DB {
	return &DB{
		host:    env.GetEnv("POSTGRES_HOST", defaultPostgresHost),
		port:    env.GetIntEnv("POSTGRES_PORT", defaultPostgresPort),
		user:    env.GetEnv("POSTGRES_USER", defaultPostgresUser),
		pass:    env.GetEnv("POSTGRES_PASSWORD", defaultPostgresPassword),
		name:    env.GetEnv("POSTGRES_DB", defaultPostgresDB),
		sslMode: env.GetEnv("POSTGRES_SSLMODE", defaultPostgresSslMode),
	}
}

func (c *DB) Host() string    { return c.host }
func (c *DB) Port() int       { return c.port }
func (c *DB) User() string    { return c.user }
func (c *DB) Pass() string    { return c.pass }
func (c *DB) Name() string    { return c.name }
func (c *DB) SSLMode() string { return c.sslMode }
