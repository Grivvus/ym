package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type ErrUserAlreadyExists struct {
	Username string
}

func (e ErrUserAlreadyExists) Error() string {
	return fmt.Sprintf("User '%v' already exists", e.Username)
}

type AuthService struct {
	queries   *db.Queries
	logger    *slog.Logger
	jwtSecret []byte
}

func NewAuthService(q *db.Queries, logger *slog.Logger, cfg *utils.Config) AuthService {
	return AuthService{
		queries:   q,
		logger:    logger,
		jwtSecret: []byte(cfg.JWTSecret),
	}
}

func (a AuthService) Register(
	ctx context.Context, user api.UserAuth,
) (api.TokenResponse, error) {
	hashed, salt := utils.HashPassword(user.Password)
	arg := db.CreateUserParams{
		Username: user.Username,
		Email:    pgtype.Text{Valid: false},
		Password: hashed,
		Salt:     salt,
	}
	createdUser, err := a.queries.CreateUser(ctx, arg)
	if err != nil {
		a.logger.Error("can't create user", "error", err)
		var retErr error
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			// duplicate key value violates unique constraint
			if pgErr.SQLState() == "23505" {
				retErr = ErrUserAlreadyExists{Username: user.Username}
			} else {
				retErr = fmt.Errorf("unknown db error: %w", err)
			}
		}
		return api.TokenResponse{}, retErr
	}

	accessToken, refreshToken, err := utils.CreateTokens(int(createdUser.ID), a.jwtSecret)

	return api.TokenResponse{
		UserId:       createdUser.ID,
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
		TokenType:    "bearer",
	}, err
}

func (a AuthService) Login(
	ctx context.Context, user api.UserAuth,
) (api.TokenResponse, error) {
	dbuser, err := a.queries.GetUserByUsername(ctx, user.Username)
	if err != nil {
		a.logger.Error("can't get user from db", "error", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return api.TokenResponse{}, NewErrNotFound("user", user.Username)
		}
		return api.TokenResponse{}, fmt.Errorf("unknown error: %w", err)
	}

	if !utils.VerifyPassword(user.Password, dbuser.Salt, dbuser.Password) {
		return api.TokenResponse{}, fmt.Errorf("wrong password")
	}

	accessToken, refreshToken, err := utils.CreateTokens(int(dbuser.ID), a.jwtSecret)

	return api.TokenResponse{
		UserId:       dbuser.ID,
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
		TokenType:    "bearer",
	}, err
}

func (a AuthService) UpdateTokens(ctx context.Context) error {
	panic("not implemented")
}

func (a AuthService) RevokeTokens(ctx context.Context) error {
	panic("not implemented")
}
