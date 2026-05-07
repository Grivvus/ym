package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/repository"
	"github.com/Grivvus/ym/internal/utils"
)

var ErrUnauthorized = errors.New("unauthorized")
var ErrSuperuserRequired = errors.New("forbidden: required superuser rights")

type AuthService struct {
	repo      repository.AuthRepository
	logger    *slog.Logger
	jwtSecret []byte
}

func NewAuthService(
	repo repository.AuthRepository, logger *slog.Logger, cfg *utils.Config,
) AuthService {
	return AuthService{
		repo:      repo,
		logger:    logger,
		jwtSecret: []byte(cfg.JWTSecret),
	}
}

func (a AuthService) Register(
	ctx context.Context, user api.UserAuth,
) (api.TokenResponse, error) {
	hashed, salt := utils.HashPassword(user.Password)
	createdUser, err := a.repo.CreateUserWithInitialRole(ctx, repository.CreateAuthUserParams{
		Username: user.Username,
		Password: hashed,
		Salt:     salt,
	})
	if err != nil {
		a.logger.Error("can't create user", "error", err)
		if errors.Is(err, repository.ErrAlreadyExists) {
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
	dbuser, err := a.repo.GetUserByUsername(ctx, user.Username)
	if err != nil {
		a.logger.Error("can't get user from db", "error", err)
		if errors.Is(err, repository.ErrNotFound) {
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

	dbuser, err := a.repo.GetUserByID(ctx, userID)
	if err != nil {
		a.logger.Error("can't get user from db", "error", err)
		if errors.Is(err, repository.ErrNotFound) {
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
	user, err := a.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUnauthorized
		}
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	if !user.IsSuperuser {
		return ErrSuperuserRequired
	}
	return nil
}
