package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Host      string `envconfig:"APPLICATION_HOST" required:"true"`
	Port      string `envconfig:"APPLICATION_PORT" required:"true"`
	JWTSecret string `envconfig:"JWT_SECRET" required:"true"`

	PostgresUser     string `envconfig:"POSTGRES_USER" required:"true"`
	PostgresHost     string `envconfig:"POSTGRES_HOST" required:"true"`
	PostgresPort     string `envconfig:"POSTGRES_PORT" required:"true"`
	PostgresPassword string `envconfig:"POSTGRES_PASSWORD" required:"true"`
	PostgresDB       string `envconfig:"POSTGRES_DB" required:"true"`

	S3Host             string `envconfig:"S3_HOST" required:"true"`
	S3Port             string `envconfig:"S3_PORT" required:"true"`
	S3AccessKey        string `envconfig:"MINIO_ACCESS_KEY" required:"true"`
	S3SecretKey        string `envconfig:"MINIO_SECRET_KEY" required:"true"`
	S3SecureConnection bool   `envconfig:"S3_SECURE" default:"false"`
}

type SMTPConfig struct {
	Host        string `envconfig:"SMTP_HOST"`
	Port        string `envconfig:"SMTP_PORT"`
	Username    string `envconfig:"SMTP_USERNAME"`
	Password    string `envconfig:"SMTP_PASSWORD"`
	FromAddress string `envconfig:"SMTP_FROM_ADDRESS"`
	FromName    string `envconfig:"SMTP_FROM_NAME"`
	TLSMode     string `envconfig:"SMTP_TLS_MODE" default:"starttls"`
}

type PasswordResetConfig struct {
	Enabled         bool          `envconfig:"PASSWORD_RESET_ENABLED" default:"false"`
	CodeSecret      string        `envconfig:"PASSWORD_RESET_CODE_SECRET"`
	CodeTTL         time.Duration `envconfig:"PASSWORD_RESET_CODE_TTL" default:"15m"`
	ResendCooldown  time.Duration `envconfig:"PASSWORD_RESET_RESEND_COOLDOWN" default:"1m"`
	MaxAttempts     int           `envconfig:"PASSWORD_RESET_MAX_ATTEMPTS" default:"5"`
	CodeLength      int           `envconfig:"PASSWORD_RESET_CODE_LENGTH" default:"6"`
	AcceptedMessage string        `ignored:"true"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, fmt.Errorf("can't process config: %w", err)
	}
	return &cfg, nil
}

func NewSMTPConfig() (*SMTPConfig, error) {
	var cfg SMTPConfig
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, fmt.Errorf("can't process smtp config: %w", err)
	}
	return &cfg, nil
}

func NewPasswordResetConfig() (*PasswordResetConfig, error) {
	var cfg PasswordResetConfig
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, fmt.Errorf("can't process password reset config: %w", err)
	}
	cfg.AcceptedMessage = "if an account with that email exists, a reset code has been sent"
	return &cfg, nil
}

func (cfg *Config) DBConnString() string {
	return fmt.Sprintf(
		"postgres://%v:%v@%v:%v/%v",
		cfg.PostgresUser, cfg.PostgresPassword,
		cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresDB,
	)
}

func (cfg SMTPConfig) IsConfigured() bool {
	return cfg.Host != "" && cfg.Port != "" && cfg.FromAddress != ""
}

func (cfg SMTPConfig) Validate() error {
	if !cfg.IsConfigured() {
		return fmt.Errorf("SMTP_HOST, SMTP_PORT, and SMTP_FROM_ADDRESS must be set")
	}

	if (cfg.Username == "") != (cfg.Password == "") {
		return fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD must be provided together")
	}

	switch strings.ToLower(cfg.TLSMode) {
	case "none", "starttls", "implicit":
		return nil
	default:
		return fmt.Errorf("SMTP_TLS_MODE must be one of: none, starttls, implicit")
	}
}

func (cfg PasswordResetConfig) Validate() error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.CodeSecret == "" {
		return fmt.Errorf("PASSWORD_RESET_CODE_SECRET must be set")
	}
	if cfg.CodeTTL <= 0 {
		return fmt.Errorf("PASSWORD_RESET_CODE_TTL must be positive")
	}
	if cfg.ResendCooldown < 0 {
		return fmt.Errorf("PASSWORD_RESET_RESEND_COOLDOWN must not be negative")
	}
	if cfg.MaxAttempts <= 0 {
		return fmt.Errorf("PASSWORD_RESET_MAX_ATTEMPTS must be positive")
	}
	if cfg.CodeLength <= 0 {
		return fmt.Errorf("PASSWORD_RESET_CODE_LENGTH must be positive")
	}
	return nil
}
