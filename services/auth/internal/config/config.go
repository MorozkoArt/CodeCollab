package config

type Config struct {
	*Server
	*DB
	*Auth
	*App
	*Mail
}

func NewConfig() *Config {
	return &Config{
		Server: NewServerConfig(),
		DB:     NewDBConfig(),
		Auth:   NewAuthConfig(),
		App:    NewAppConfig(),
		Mail:   NewMailConfig(),
	}
}
