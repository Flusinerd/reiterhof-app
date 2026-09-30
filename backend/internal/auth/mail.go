package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
)

// Message is a plain-text mail.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Mailer sends mail. Implementations: SMTPMailer (production), LogMailer (development).
type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// NewMailer returns an SMTPMailer if REITERHOF_SMTP_HOST is set, else a LogMailer that
// writes the mail (including the login link) to the log.
func NewMailer(cfg config.Auth, log *slog.Logger) Mailer {
	if cfg.SMTPHost == "" {
		if log == nil {
			log = slog.Default()
		}
		log.Warn("REITERHOF_SMTP_HOST not set: login mails are only written to the log")
		return LogMailer{Log: log}
	}
	return &SMTPMailer{Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom}
}

// LogMailer logs mails instead of sending them (development only).
type LogMailer struct{ Log *slog.Logger }

// Send implements Mailer.
func (l LogMailer) Send(_ context.Context, m Message) error {
	l.Log.Info("mail (not sent, no SMTP configured)", "to", m.To, "subject", m.Subject, "body", m.Body)
	return nil
}

// SMTPMailer sends through an SMTP server with net/smtp. Port 465 uses implicit TLS,
// other ports use STARTTLS when the server offers it (required when credentials are set).
type SMTPMailer struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string // "login@example.org" or "Stallfunk <login@example.org>"
}

// Send implements Mailer.
func (s *SMTPMailer) Send(ctx context.Context, m Message) error {
	from, err := mail.ParseAddress(s.From)
	if err != nil {
		return fmt.Errorf("smtp: invalid REITERHOF_SMTP_FROM: %w", err)
	}
	to, err := mail.ParseAddress(m.To)
	if err != nil {
		return fmt.Errorf("smtp: invalid recipient: %w", err)
	}
	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	if s.Port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("smtp: dial: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp: hello: %w", err)
	}
	defer c.Close()
	if s.Port != 465 {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("smtp: starttls: %w", err)
			}
		}
	}
	if s.User != "" {
		// PlainAuth refuses to send credentials over an unencrypted connection (except localhost).
		if err := c.Auth(smtp.PlainAuth("", s.User, s.Password, s.Host)); err != nil {
			return fmt.Errorf("smtp: auth: %w", err)
		}
	}
	if err := c.Mail(from.Address); err != nil {
		return fmt.Errorf("smtp: mail from: %w", err)
	}
	if err := c.Rcpt(to.Address); err != nil {
		return fmt.Errorf("smtp: rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp: data: %w", err)
	}
	if _, err := w.Write(buildMessage(from, to, m)); err != nil {
		return fmt.Errorf("smtp: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp: close data: %w", err)
	}
	return c.Quit()
}

func buildMessage(from, to *mail.Address, m Message) []byte {
	// Addresses come from mail.ParseAddress, so they contain no CR/LF.
	var b strings.Builder
	b.WriteString("From: " + from.String() + "\r\n")
	b.WriteString("To: " + to.String() + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", m.Subject) + "\r\n")
	b.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(strings.ReplaceAll(strings.ReplaceAll(m.Body, "\r\n", "\n"), "\n", "\r\n"))
	b.WriteString("\r\n")
	return []byte(b.String())
}
