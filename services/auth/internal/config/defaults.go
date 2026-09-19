package config

import "time"

const (
	// Server
	defaultServerPort = 8080
	defaultServerHost = "0.0.0.0"

	// Database
	defaultDBHost    = "localhost"
	defaultDBPort    = 5432
	defaultDBUser    = "app_user"
	defaultDBName    = "codecollab"
	defaultDBSSLMode = "disable"

	// Auth
	defaultTokenExpiry = 24 * time.Hour

	// App
	defaultAppEnv = "development"
)
