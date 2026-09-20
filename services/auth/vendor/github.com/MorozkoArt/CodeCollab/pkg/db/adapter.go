package db

type DbConfig interface {
	Host() string
	Port() int
	User() string
	Pass() string
	Name() string
	SSLMode() string
}
