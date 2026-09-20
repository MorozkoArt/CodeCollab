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
		host:    env.Get("POSTGRES_HOST", ""),
		port:    env.GetInt("POSTGRES_PORT", defaultPostgresPort),
		user:    env.Get("POSTGRES_USER", ""),
		pass:    env.Get("POSTGRES_PASSWORD", ""),
		name:    env.Get("POSTGRES_DB", ""),
		sslMode: env.Get("POSTGRES_SSLMODE", defaultPostgresSslMode),
	}
}

func (c *DB) Host() string    { return c.host }
func (c *DB) Port() int       { return c.port }
func (c *DB) User() string    { return c.user }
func (c *DB) Pass() string    { return c.pass }
func (c *DB) Name() string    { return c.name }
func (c *DB) SSLMode() string { return c.sslMode }
