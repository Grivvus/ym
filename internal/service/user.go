package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5"
)

type ErrNoSuchUser struct {
	id int
}

func (e ErrNoSuchUser) Error() string {
	return fmt.Sprintf("user with id: '%v' not found", e.id)
}

type UserService struct {
	queries *db.Queries
}

func (u *UserService) GetUserByID(
	ctx context.Context, userId int,
) (api.UserReturn, error) {
	var ret api.UserReturn

	user, err := u.queries.GetUserByID(ctx, int32(userId))
	if err != nil {
		slog.Error("GetuserByID", "err", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, ErrNoSuchUser{id: userId}
		}
		return ret, fmt.Errorf("unkown server error: %w", err)
	}

	ret.Username = user.Username
	if user.Email.Valid {
		ret.Email = &user.Email.String
	} else {
		ret.Email = nil
	}

	return ret, nil
}
