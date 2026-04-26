package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/Grivvus/ym/internal/utils"
)

const smtpDialTimeout = 10 * time.Second

type SMTPMailer struct {
	logger *slog.Logger
	cfg    utils.SMTPConfig
}

func NewSMTPMailer(cfg utils.SMTPConfig, logger *slog.Logger) (*SMTPMailer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &SMTPMailer{
		logger: logger,
		cfg:    cfg,
	}, nil
}

func (m SMTPMailer) SendPasswordResetCode(
	ctx context.Context, recipient string, code string, ttl time.Duration,
) error {
	m.logger.Info("send password reset code")
	message := buildMessage(
		formatAddress(m.cfg.FromName, m.cfg.FromAddress),
		recipient,
		"Password reset code",
		fmt.Sprintf(
			"Your password reset code is: %s\r\nThis code expires in %s.\r\nIf you did not request a password reset, you can ignore this email.\r\n",
			code,
			ttl.Round(time.Second),
		),
	)

	return m.send(ctx, recipient, []byte(message))
}

func (m SMTPMailer) send(
	ctx context.Context, recipient string, payload []byte,
) error {
	client, closeConn, err := m.connect(ctx)
	if err != nil {
		return err
	}
	defer closeConn()

	if m.cfg.Username != "" {
		auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate with SMTP server: %w", err)
		}
	}

	if err := client.Mail(m.cfg.FromAddress); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP data writer: %w", err)
	}
	if _, err := writer.Write(payload); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close SMTP data writer: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("quit SMTP session: %w", err)
	}

	return nil
}

func (m SMTPMailer) connect(
	ctx context.Context,
) (_ *smtp.Client, closeConn func(), err error) {
	address := net.JoinHostPort(m.cfg.Host, m.cfg.Port)
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: m.cfg.Host}
	tlsMode := strings.ToLower(m.cfg.TLSMode)

	switch tlsMode {
	case "implicit":
		dialer := &net.Dialer{Timeout: smtpDialTimeout}
		conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("dial SMTP server with implicit TLS: %w", err)
		}
		client, err := smtp.NewClient(conn, m.cfg.Host)
		if err != nil {
			_ = conn.Close()
			return nil, nil, fmt.Errorf("create SMTP client: %w", err)
		}
		return client, func() {
			_ = client.Close()
		}, nil
	default:
		dialer := &net.Dialer{Timeout: smtpDialTimeout}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, nil, fmt.Errorf("dial SMTP server: %w", err)
		}
		client, err := smtp.NewClient(conn, m.cfg.Host)
		if err != nil {
			_ = conn.Close()
			return nil, nil, fmt.Errorf("create SMTP client: %w", err)
		}

		if tlsMode == "starttls" {
			if ok, _ := client.Extension("STARTTLS"); !ok {
				_ = client.Close()
				return nil, nil, fmt.Errorf("SMTP server does not support STARTTLS")
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				_ = client.Close()
				return nil, nil, fmt.Errorf("upgrade SMTP connection with STARTTLS: %w", err)
			}
		}

		return client, func() {
			_ = client.Close()
		}, nil
	}
}

func buildMessage(from string, to string, subject string, body string) string {
	return strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
}

func formatAddress(name string, address string) string {
	addr := mail.Address{Address: address}
	if strings.TrimSpace(name) != "" {
		addr.Name = name
	}
	return addr.String()
}
