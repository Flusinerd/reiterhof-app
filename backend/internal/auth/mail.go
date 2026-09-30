package auth

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
)

// Message is a mail: Body is the plain text, HTML the optional branded version (see
// MailContent). With HTML the mail is multipart/alternative and carries the logo inline.
type Message struct {
	To      string
	Subject string
	Body    string
	HTML    string
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
	var b bytes.Buffer
	b.WriteString("From: " + from.String() + "\r\n")
	b.WriteString("To: " + to.String() + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", m.Subject) + "\r\n")
	b.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	if m.HTML == "" {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		writeQP(&b, m.Body)
		return b.Bytes()
	}
	// multipart/alternative: text first, the preferred HTML last; the HTML and its logo
	// sit in multipart/related so the img src="cid:..." resolves.
	alt := multipart.NewWriter(&b)
	b.WriteString("Content-Type: multipart/alternative; boundary=" + alt.Boundary() + "\r\n\r\n")
	text, _ := alt.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	writeQP(text, m.Body)
	var rel bytes.Buffer
	relW := multipart.NewWriter(&rel)
	html, _ := relW.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/html; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	writeQP(html, m.HTML)
	logo, _ := relW.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"image/png"},
		"Content-Transfer-Encoding": {"base64"},
		"Content-ID":                {"<" + mailLogoCID + ">"},
		"Content-Disposition":       {`inline; filename="stallfunk.png"`},
	})
	writeBase64(logo, mailLogo)
	_ = relW.Close()
	relPart, _ := alt.CreatePart(textproto.MIMEHeader{
		"Content-Type": {`multipart/related; type="text/html"; boundary=` + relW.Boundary()},
	})
	_, _ = relPart.Write(rel.Bytes())
	_ = alt.Close()
	return b.Bytes()
}

// writeQP writes s quoted-printable with CRLF line ends (keeps lines under the SMTP limit).
func writeQP(w interface{ Write([]byte) (int, error) }, s string) {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\n", "\r\n")
	qp := quotedprintable.NewWriter(w)
	_, _ = qp.Write([]byte(s))
	_ = qp.Close()
	_, _ = w.Write([]byte("\r\n"))
}

// writeBase64 writes data base64 encoded in lines of 76 characters.
func writeBase64(w interface{ Write([]byte) (int, error) }, data []byte) {
	enc := base64.StdEncoding.EncodeToString(data)
	for len(enc) > 76 {
		_, _ = w.Write([]byte(enc[:76] + "\r\n"))
		enc = enc[76:]
	}
	_, _ = w.Write([]byte(enc + "\r\n"))
}
