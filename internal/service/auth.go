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

var secret = []byte("hackme")

type ErrUserAlreadyExists struct {
	Username string
}

func (e ErrUserAlreadyExists) Error() string {
	return fmt.Sprintf("User '%v' already exists", e.Username)
}

type AuthService struct {
	queries *db.Queries
}

func NewAuthService(q *db.Queries) AuthService {
	return AuthService{
		queries: q,
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
		slog.Error("AuthService.Register", "error", err)
		var retErr error
		if pgErr, ok := err.(*pgconn.PgError); ok {
			// duplicate key value violates unique constraint
			if pgErr.SQLState() == "23505" {
				retErr = ErrUserAlreadyExists{Username: user.Username}
			} else {
				retErr = fmt.Errorf("Unkown db error: %w", err)
			}
		} else {
			retErr = fmt.Errorf("Unkown error: %w", err)
		}
		return api.TokenResponse{}, retErr
	}

	accessToken, refreshToken, err := utils.CreateTokens(int(createdUser.ID), secret)

	return api.TokenResponse{
		UserId:       int(createdUser.ID),
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
		TokenType:    "bearer",
	}, nil
}

func (a AuthService) Login(
	ctx context.Context, user api.UserAuth,
) (api.TokenResponse, error) {
	dbuser, err := a.queries.GetUserByUsername(ctx, user.Username)
	if err != nil {
		slog.Error("AuthService.Login", "error", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return api.TokenResponse{}, NewErrNotFound("user", user.Username)
		}
		return api.TokenResponse{}, fmt.Errorf("Unkown error: %w", err)
	}

	if !utils.VerifyPassword(user.Password, dbuser.Salt, dbuser.Password) {
		return api.TokenResponse{}, fmt.Errorf("wrong password")
	}

	accessToken, refreshToken, err := utils.CreateTokens(int(dbuser.ID), secret)

	return api.TokenResponse{
		UserId:       int(dbuser.ID),
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
		TokenType:    "bearer",
	}, nil
}
