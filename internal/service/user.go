package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ErrNoSuchUser struct {
	identifier any
}

func (e ErrNoSuchUser) Error() string {
	return fmt.Sprintf("user %v not found", e.identifier)
}

type UserService struct {
	queries *db.Queries
}

func NewUserService(q *db.Queries) UserService {
	return UserService{
		queries: q,
	}
}

func (u *UserService) GetUserByID(
	ctx context.Context, userID int,
) (api.UserReturn, error) {
	var ret api.UserReturn

	user, err := u.queries.GetUserByID(ctx, int32(userID))
	if err != nil {
		slog.Error("GetuserByID", "err", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, ErrNoSuchUser{identifier: userID}
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

func (u *UserService) ChangeUser(
	ctx context.Context,
	userID int,
	newUserParams api.UserUpdate,
) (api.UserReturn, error) {
	updateParamsDB := db.UpdateUserParams{
		ID:       int32(userID),
		Username: newUserParams.NewUsername,
	}
	if newUserParams.NewEmail != "" {
		updateParamsDB.Email = pgtype.Text{
			String: newUserParams.NewEmail,
			Valid:  true,
		}
	} else {
		updateParamsDB.Email = pgtype.Text{
			Valid: false,
		}
	}

	updatedUser, err := u.queries.UpdateUser(ctx, updateParamsDB)
	var ret api.UserReturn
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, ErrNoSuchUser{identifier: userID}
		}
		return ret, fmt.Errorf("unkown server error: %w", err)
	}

	ret.Username = updatedUser.Username
	if updatedUser.Email.Valid {
		ret.Email = &updatedUser.Email.String
	} else {
		ret.Email = nil
	}

	return ret, nil
}
