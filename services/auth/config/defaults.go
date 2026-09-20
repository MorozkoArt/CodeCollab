package config

import "time"

const (
	// Server
	defaultServerPort = 8080
	defaultServerHost = "0.0.0.0"

	// Database
	defaultPostgresPort    = 5432
	defaultPostgresSslMode = "disable"

	// Auth
	defaultTokenExpiry = 24 * time.Hour

	// App
	defaultAppEnv = "development"
)
