package service

import (
	"context"
	"fmt"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/jackc/pgx/v5/pgtype"
)

var secret = []byte("hackme")

type AuthService struct {
	queries *db.Queries
}

func (a AuthService) Register(
	ctx context.Context, user api.UserAuth,
) (api.TokenResponse, error) {
	arg := db.CreateUserParams{
		Username: user.Username,
		Email:    pgtype.Text{Valid: false},
		Password: utils.HashPassword(user.Password),
	}
	createdUser, err := a.queries.CreateUser(ctx, arg)
	if err != nil {
		// should check if error because user with this username already exists
		panic("not implemented")
		return api.TokenResponse{}, fmt.Errorf("can't create user, got error: '%w'", err)
	}

	accessToken, refreshToken, err := utils.CreateTokens(int(createdUser.ID), secret)

	return api.TokenResponse{
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
		panic("not implemented")
	}

	if utils.HashPassword(user.Password) != dbuser.Password {
		return api.TokenResponse{}, fmt.Errorf("wrong password")
	}

	accessToken, refreshToken, err := utils.CreateTokens(int(dbuser.ID), secret)

	return api.TokenResponse{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
		TokenType:    "bearer",
	}, nil
}
