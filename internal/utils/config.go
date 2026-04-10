package utils

import (
	"fmt"

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

	TLSCertFile string `envconfig:"APPLICATION_TLS_CERT_FILE"`
	TLSKeyFile  string `envconfig:"APPLICATION_TLS_KEY_FILE"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, fmt.Errorf("can't process config: %w", err)
	}
	return &cfg, nil
}

func (cfg *Config) DBConnString() string {
	return fmt.Sprintf(
		"postgres://%v:%v@%v:%v/%v",
		cfg.PostgresUser, cfg.PostgresPassword,
		cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresDB,
	)
}
