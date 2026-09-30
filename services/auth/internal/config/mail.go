package config

import (
	"errors"
	"fmt"
	"net/mail"

	"github.com/MorozkoArt/CodeCollab/pkg/env"
	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/enum"
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

func (c *Mail) Validate() error {
	if c.host == "" {
		return errors.New("SMTP_HOST is required")
	}
	switch c.tls {
	case enum.SMTPTLSStartTLS, enum.SMTPTLSImplicit, enum.SMTPTLSNone:
	default:
		return fmt.Errorf("SMTP_TLS must be %s, %s or %s, got %q", enum.SMTPTLSStartTLS, enum.SMTPTLSImplicit, enum.SMTPTLSNone, c.tls)
	}
	if _, err := mail.ParseAddress(c.from); err != nil {
		return fmt.Errorf("SMTP_FROM is invalid: %w", err)
	}
	return nil
}
