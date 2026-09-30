package config

import (
	"net/mail"

	"github.com/MorozkoArt/CodeCollab/pkg/env"
)

type Mail struct {
	host, user, pass, from, tls string
	port                        int
}

func NewMailConfig() *Mail {
	return &Mail{
		host: env.GetEnv("SMTP_HOST", ""),
		port: env.GetIntEnv("SMTP_PORT", defaultSMTPPort),
		user: env.GetEnv("SMTP_USER", ""),
		pass: env.GetEnv("SMTP_PASSWORD", ""),
		from: env.GetEnv("SMTP_FROM", ""),
		tls:  env.GetEnv("SMTP_TLS", defaultSMTPTLS),
	}
}

func (c *Mail) SMTPHost() string     { return c.host }
func (c *Mail) SMTPPort() int        { return c.port }
func (c *Mail) SMTPUser() string     { return c.user }
func (c *Mail) SMTPPassword() string { return c.pass }
func (c *Mail) SMTPTLS() string      { return c.tls }
func (c *Mail) From() string         { return c.from }

func (c *Mail) FromAddress() string {
	if a, err := mail.ParseAddress(c.from); err == nil {
		return a.Address
	}
	return c.from
}
