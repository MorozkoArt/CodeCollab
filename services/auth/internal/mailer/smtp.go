package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"time"

	"github.com/MorozkoArt/CodeCollab/services/auth/internal/config"
	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/enum"
)

const (
	dialTimeout    = 10 * time.Second
	sessionTimeout = 8 * time.Second
)

type SMTP struct {
	cfg     *config.Mail
	appName string
}

func NewSMTP(cfg *config.Mail, appName string) *SMTP {
	return &SMTP{cfg: cfg, appName: appName}
}

func (m *SMTP) SendCode(ctx context.Context, to, purpose, code string, ttl time.Duration) error {
	rcpt, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("parse recipient: %w", err)
	}

	now := time.Now()
	subject, text, html, err := renderCode(codeParams{
		AppName: m.appName, Purpose: purpose, Code: code, TTL: ttl, Now: now,
	})
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	raw, err := message{
		From: m.cfg.From(), FromAddr: m.cfg.FromAddress(), To: rcpt.Address,
		Subject: subject, Text: text, HTML: html, Date: now,
	}.bytes()
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	return m.deliver(ctx, rcpt.Address, raw)
}

func (m *SMTP) deliver(ctx context.Context, to string, msg []byte) error {
	c, err := m.connect(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	if err = m.authenticate(c); err != nil {
		return err
	}
	if err = m.transfer(c, to, msg); err != nil {
		return err
	}

	_ = c.Quit()
	return nil
}

func (m *SMTP) connect(ctx context.Context) (*smtp.Client, error) {
	host := m.cfg.SMTPHost()
	addr := net.JoinHostPort(host, strconv.Itoa(m.cfg.SMTPPort()))
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: dialTimeout}

	var (
		conn net.Conn
		err  error
	)
	if m.cfg.SMTPTLS() == enum.SMTPTLSImplicit {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsCfg}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("smtp dial: %w", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(sessionTimeout)
	}
	_ = conn.SetDeadline(deadline)

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("smtp client: %w", err)
	}

	if m.cfg.SMTPTLS() == enum.SMTPTLSStartTLS {
		if err = c.StartTLS(tlsCfg); err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("smtp starttls: %w", err)
		}
	}
	return c, nil
}

func (m *SMTP) authenticate(c *smtp.Client) error {
	if m.cfg.SMTPUser() == "" {
		return nil
	}
	auth := smtp.PlainAuth("", m.cfg.SMTPUser(), m.cfg.SMTPPassword(), m.cfg.SMTPHost())
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	return nil
}

func (m *SMTP) transfer(c *smtp.Client, to string, msg []byte) error {
	if err := c.Mail(m.cfg.FromAddress()); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return nil
}
