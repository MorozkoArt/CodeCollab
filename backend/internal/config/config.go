package config

type Config struct {
	*ServerConfig
	*DBConfig
	*AuthConfig
	*AppConfig
}

func NewConfig() *Config {
	return &Config{
		ServerConfig: NewServerConfig(),
		DBConfig:     NewDBConfig(),
		AuthConfig:   NewAuthConfig(),
		AppConfig:    NewAppConfig(),
	}
}
