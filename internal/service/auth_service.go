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

var ErrUnauthorized = errors.New("unauthorized")
var ErrSuperuserRequired = errors.New("forbidden: required superuser rights")

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
	usersCnt, err := a.queries.GetUserCount(ctx)
	if err != nil {
		return api.TokenResponse{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	hashed, salt := utils.HashPassword(user.Password)
	arg := db.CreateUserParams{
		Username:    user.Username,
		Email:       pgtype.Text{Valid: false},
		Password:    hashed,
		Salt:        salt,
		IsSuperuser: usersCnt == 0,
	}
	createdUser, err := a.queries.CreateUser(ctx, arg)
	if err != nil {
		a.logger.Error("can't create user", "error", err)
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return api.TokenResponse{}, NewErrAlreadyExists("user", user.Username)
		}
		return api.TokenResponse{}, fmt.Errorf("%w cause: %w", ErrUnknownDBError, err)
	}

	accessToken, refreshToken, err := utils.CreateTokensWithRefreshVersion(
		int(createdUser.ID), createdUser.RefreshVersion, a.jwtSecret,
	)

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
		return api.TokenResponse{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	if !utils.VerifyPassword(user.Password, dbuser.Salt, dbuser.Password) {
		return api.TokenResponse{}, ErrUnauthorized
	}

	accessToken, refreshToken, err := utils.CreateTokensWithRefreshVersion(
		int(dbuser.ID), dbuser.RefreshVersion, a.jwtSecret,
	)

	return api.TokenResponse{
		UserId:       dbuser.ID,
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
		TokenType:    "bearer",
	}, err
}

func (a AuthService) UpdateTokens(
	ctx context.Context, refreshToken string,
) (api.TokenResponse, error) {
	userID, refreshVersion, err := utils.ParseRefreshTokenWithVersion(
		refreshToken, a.jwtSecret,
	)
	if err != nil {
		return api.TokenResponse{}, ErrUnauthorized
	}

	dbuser, err := a.queries.GetUserByID(ctx, userID)
	if err != nil {
		a.logger.Error("can't get user from db", "error", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return api.TokenResponse{}, ErrUnauthorized
		}
		return api.TokenResponse{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	if refreshVersion != dbuser.RefreshVersion {
		return api.TokenResponse{}, ErrUnauthorized
	}

	accessToken, newRefreshToken, err := utils.CreateTokensWithRefreshVersion(
		int(dbuser.ID), dbuser.RefreshVersion, a.jwtSecret,
	)
	if err != nil {
		return api.TokenResponse{}, fmt.Errorf("can't create tokens: %w", err)
	}

	return api.TokenResponse{
		UserId:       dbuser.ID,
		RefreshToken: newRefreshToken,
		AccessToken:  accessToken,
		TokenType:    "bearer",
	}, nil
}

func (a AuthService) RevokeTokens(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (a AuthService) AuthorizeSuperuser(ctx context.Context, userID int32) error {
	user, err := a.queries.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	if !user.IsSuperuser {
		return ErrSuperuserRequired
	}
	return nil
}
