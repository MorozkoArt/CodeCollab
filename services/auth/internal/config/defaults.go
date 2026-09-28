package config

import "time"

const (
	// Server
	defaultServerPort = 8080
	defaultServerHost = "0.0.0.0"
	defaultGRPCPort   = 9091

	// Database
	defaultPostgresHost     = "postgres"
	defaultPostgresPort     = 5432
	defaultPostgresUser     = "admin"
	defaultPostgresPassword = "11111111"
	defaultPostgresDB       = "codecollab"
	defaultPostgresSslMode  = "disable"

	// Auth
	defaultTokenExpiry = 24 * time.Hour

	// App
	defaultAppEnv = "development"
)
