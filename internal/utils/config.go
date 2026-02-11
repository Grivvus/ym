package utils

import (
	"fmt"
	"os"
)

type Config struct {
	Host      string
	Port      string
	JWTSecret string

	PostgresUser     string
	PostgresHost     string
	PostgresPort     string
	PostgresPassword string
	PostgresDB       string

	S3Host             string
	S3Port             string
	S3AccessKey        string
	S3SecretKey        string
	S3SecureConnection bool
}

func NewConfig() (*Config, error) {
	host, ok := os.LookupEnv("APPLICATION_HOST")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "APPLICATION_HOST")
	}
	port, ok := os.LookupEnv("APPLICATION_PORT")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "APPLICATION_PORT")
	}
	jwtsecret, ok := os.LookupEnv("JWT_SECRET")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "JWT_SECRET")
	}
	postgresUser, ok := os.LookupEnv("POSTGRES_USER")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "POSTGRES_USER")
	}
	postgresHost, ok := os.LookupEnv("POSTGRES_HOST")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "POSTGRES_HOST")
	}
	postgresPort, ok := os.LookupEnv("POSTGRES_PORT")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "POSTGRES_PORT")
	}
	postgresPassword, ok := os.LookupEnv("POSTGRES_PASSWORD")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "POSTGRES_PASSWORD")
	}
	postgresDB, ok := os.LookupEnv("POSTGRES_DB")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "POSTGRES_DB")
	}
	s3Host, ok := os.LookupEnv("S3_HOST")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "S3_HOST")
	}
	s3Port := os.Getenv("S3_PORT")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "S3_PORT")
	}
	minioAccess := os.Getenv("MINIO_ACCESS_KEY")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "MINIO_ACCESS_KEY")
	}
	minioSecret := os.Getenv("MINIO_SECRET_KEY")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "MINIO_SECRET_KEY")
	}
	fmt.Println(minioAccess, minioSecret)

	return &Config{
		Host:               host,
		Port:               port,
		JWTSecret:          jwtsecret,
		PostgresUser:       postgresUser,
		PostgresHost:       postgresHost,
		PostgresPort:       postgresPort,
		PostgresPassword:   postgresPassword,
		PostgresDB:         postgresDB,
		S3Host:             s3Host,
		S3Port:             s3Port,
		S3AccessKey:        minioAccess,
		S3SecretKey:        minioSecret,
		S3SecureConnection: false,
	}, nil
}

func (cfg *Config) DBConnString() string {
	return fmt.Sprintf(
		"postgres://%v:%v@%v:%v/%v",
		cfg.PostgresUser, cfg.PostgresPassword,
		cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresDB,
	)
}
