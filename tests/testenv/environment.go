package testenv

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresPort   = "5432/tcp"
	storagePort    = "9000/tcp"
	defaultDBName  = "ym"
	defaultDBUser  = "ym"
	defaultDBPass  = "ym"
	defaultS3User  = "minioadmin"
	defaultS3Pass  = "minioadmin"
	retryInterval  = 300 * time.Millisecond
	startupTimeout = 2 * time.Minute
)

type Environment struct {
	Config        utils.Config
	DB            *pgxpool.Pool
	Queries       *db.Queries
	Storage       storage.Storage
	Postgres      *testcontainers.DockerContainer
	ObjectStorage *testcontainers.DockerContainer
}

func Start(ctx context.Context) (_ *Environment, err error) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	env := &Environment{}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("bootstrap integration containers: %v", recovered)
		}

		if err != nil {
			_ = env.Close(context.Background())
		}
	}()

	env.Postgres, err = startPostgres(ctx)
	if err != nil {
		return nil, err
	}

	env.ObjectStorage, err = startObjectStorage(ctx)
	if err != nil {
		return nil, err
	}

	cfg, connURI, err := buildConfig(ctx, env.Postgres, env.ObjectStorage)
	if err != nil {
		return nil, err
	}
	env.Config = cfg

	if err = runMigrations(ctx, connURI); err != nil {
		return nil, err
	}

	env.DB, err = connectPool(ctx, connURI)
	if err != nil {
		return nil, err
	}

	env.Queries = db.New(env.DB)
	env.Storage, err = connectStorage(ctx, env.Config, logger)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func (e *Environment) Close(ctx context.Context) error {
	var err error

	if e.DB != nil {
		e.DB.Close()
	}

	if e.ObjectStorage != nil {
		err = errors.Join(err, e.ObjectStorage.Terminate(ctx))
	}

	if e.Postgres != nil {
		err = errors.Join(err, e.Postgres.Terminate(ctx))
	}

	return err
}

func startPostgres(ctx context.Context) (*testcontainers.DockerContainer, error) {
	container, err := testcontainers.Run(
		ctx,
		imageName("TEST_POSTGRES_IMAGE", "postgres:17-alpine"),
		testcontainers.WithEnv(map[string]string{
			"POSTGRES_DB":       defaultDBName,
			"POSTGRES_USER":     defaultDBUser,
			"POSTGRES_PASSWORD": defaultDBPass,
		}),
		testcontainers.WithExposedPorts(postgresPort),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(startupTimeout),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	return container, nil
}

func startObjectStorage(ctx context.Context) (*testcontainers.DockerContainer, error) {
	container, err := testcontainers.Run(
		ctx,
		imageName("TEST_MINIO_IMAGE", "minio/minio:latest"),
		testcontainers.WithEnv(map[string]string{
			"MINIO_ROOT_USER":     defaultS3User,
			"MINIO_ROOT_PASSWORD": defaultS3Pass,
		}),
		testcontainers.WithCmd("server", "/data", "--console-address", ":9001"),
		testcontainers.WithExposedPorts(storagePort),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/minio/health/live").
				WithPort(storagePort).
				WithStartupTimeout(startupTimeout),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start minio container: %w", err)
	}

	return container, nil
}

func buildConfig(
	ctx context.Context,
	postgresContainer *testcontainers.DockerContainer,
	objectStorageContainer *testcontainers.DockerContainer,
) (utils.Config, string, error) {
	postgresHost, err := postgresContainer.Host(ctx)
	if err != nil {
		return utils.Config{}, "", fmt.Errorf("get postgres host: %w", err)
	}

	postgresMappedPort, err := postgresContainer.MappedPort(ctx, postgresPort)
	if err != nil {
		return utils.Config{}, "", fmt.Errorf("get postgres port: %w", err)
	}

	storageHost, err := objectStorageContainer.Host(ctx)
	if err != nil {
		return utils.Config{}, "", fmt.Errorf("get storage host: %w", err)
	}

	storageMappedPort, err := objectStorageContainer.MappedPort(ctx, storagePort)
	if err != nil {
		return utils.Config{}, "", fmt.Errorf("get storage port: %w", err)
	}

	cfg := utils.Config{
		Host:               "127.0.0.1",
		Port:               "0",
		JWTSecret:          "integration-secret",
		PostgresUser:       defaultDBUser,
		PostgresHost:       postgresHost,
		PostgresPort:       postgresMappedPort.Port(),
		PostgresPassword:   defaultDBPass,
		PostgresDB:         defaultDBName,
		S3Host:             storageHost,
		S3Port:             storageMappedPort.Port(),
		S3AccessKey:        defaultS3User,
		S3SecretKey:        defaultS3Pass,
		S3SecureConnection: false,
	}

	return cfg, postgresConnectionString(cfg), nil
}

func runMigrations(ctx context.Context, connURI string) error {
	migrationsDir, err := resolveMigrationsDir()
	if err != nil {
		return err
	}

	sqlDB, err := sql.Open("pgx", connURI)
	if err != nil {
		return fmt.Errorf("open sql connection for migrations: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database before migrations: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, sqlDB, migrationsDir); err != nil {
		return fmt.Errorf("apply goose migrations: %w", err)
	}

	return nil
}

func connectPool(ctx context.Context, connURI string) (*pgxpool.Pool, error) {
	var lastErr error
	for {
		pool, err := pgxpool.New(ctx, connURI)
		if err == nil {
			pingErr := pool.Ping(ctx)
			if pingErr == nil {
				return pool, nil
			}

			lastErr = pingErr
			pool.Close()
		} else {
			lastErr = err
		}

		if ctx.Err() != nil {
			return nil, fmt.Errorf("connect to postgres: %w", errors.Join(lastErr, ctx.Err()))
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect to postgres: %w", errors.Join(lastErr, ctx.Err()))
		case <-time.After(retryInterval):
		}
	}
}

func connectStorage(ctx context.Context, cfg utils.Config, logger *slog.Logger) (storage.Storage, error) {
	var lastErr error
	for {
		st, err := storage.New(ctx, cfg, logger)
		if err == nil {
			return st, nil
		}

		lastErr = err
		if ctx.Err() != nil {
			return nil, fmt.Errorf("connect to storage: %w", errors.Join(lastErr, ctx.Err()))
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect to storage: %w", errors.Join(lastErr, ctx.Err()))
		case <-time.After(retryInterval):
		}
	}
}

func resolveMigrationsDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("resolve migrations dir: runtime caller is unavailable")
	}

	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return filepath.Join(projectRoot, "db", "migrations"), nil
}

func postgresConnectionString(cfg utils.Config) string {
	return fmt.Sprintf("%s?sslmode=disable", cfg.DBConnString())
}

func imageName(envKey, fallback string) string {
	if override := os.Getenv(envKey); override != "" {
		return override
	}

	return fallback
}
