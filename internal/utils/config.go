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

	S3Host         string
	S3Port         string
	S3RootUser     string
	S3RootPassword string
}

func NewConfig() (*Config, error) {
	host, ok := os.LookupEnv("APPLICATION_HOST")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "APPLICATION_HOST")
	}
	port, ok := os.LookupEnv("APPLICATION_PORT")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "APPLICATION_HOST")
	}
	jwtsecret, ok := os.LookupEnv("JWT_SECRET")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "APPLICATION_HOST")
	}
	postgresUser, ok := os.LookupEnv("POSTGRES_USER")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "APPLICATION_HOST")
	}
	postgresHost, ok := os.LookupEnv("POSTGRES_HOST")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "APPLICATION_HOST")
	}
	postgresPort, ok := os.LookupEnv("POSTGRES_PORT")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "APPLICATION_HOST")
	}
	postgresPassword, ok := os.LookupEnv("POSTGRES_PASSWORD")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "APPLICATION_HOST")
	}
	postgresDB, ok := os.LookupEnv("POSTGRES_DB")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "APPLICATION_HOST")
	}
	s3Host, ok := os.LookupEnv("S3_HOST")
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %v", "APPLICATION_HOST")
	}
	s3Port := os.Getenv("S3_PORT")
	s3RootUser := os.Getenv("S3_ROOT_USER")
	s3RootPassword := os.Getenv("S3_ROOT_PASSWORD")

	return &Config{
		Host:             host,
		Port:             port,
		JWTSecret:        jwtsecret,
		PostgresUser:     postgresUser,
		PostgresHost:     postgresHost,
		PostgresPort:     postgresPort,
		PostgresPassword: postgresPassword,
		PostgresDB:       postgresDB,
		S3Host:           s3Host,
		S3Port:           s3Port,
		S3RootUser:       s3RootUser,
		S3RootPassword:   s3RootPassword,
	}, nil
}
