package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/MorozkoArt/CodeCollab/services/auth/internal/config"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/enum"
)

type SMTP struct{ cfg *config.Mail }

func NewSMTP(cfg *config.Mail) *SMTP { return &SMTP{cfg: cfg} }

func (m *SMTP) SendCode(ctx context.Context, to, purpose, code string, ttl time.Duration) error {
	rcpt, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("parse recipient: %w", err)
	}

	subject := "Код для входа"
	if purpose == enum.PurposeRegister {
		subject = "Код подтверждения регистрации"
	}

	body := fmt.Sprintf("Ваш код: %s\r\n\r\nОн действует %d минут. Никому его не сообщайте.\r\n"+
		"Если вы не запрашивали код, просто проигнорируйте это письмо.\r\n", code, int(ttl.Minutes()))

	msg := strings.Join([]string{
		"From: " + m.cfg.From(),
		"To: " + rcpt.Address,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")

	return m.send(ctx, rcpt.Address, []byte(msg))
}

func (m *SMTP) send(ctx context.Context, to string, msg []byte) error {
	host := m.cfg.SMTPHost()
	addr := net.JoinHostPort(host, strconv.Itoa(m.cfg.SMTPPort()))
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	var (
		conn net.Conn
		err  error
	)
	if m.cfg.SMTPTLS() == "implicit" {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsCfg}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}
	_ = conn.SetDeadline(deadline)

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if m.cfg.SMTPTLS() == "starttls" {
		if err = c.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if m.cfg.SMTPUser() != "" {
		if err = c.Auth(smtp.PlainAuth("", m.cfg.SMTPUser(), m.cfg.SMTPPassword(), host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err = c.Mail(m.cfg.FromAddress()); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err = c.Rcpt(to); err != nil {
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
	return c.Quit()
}
