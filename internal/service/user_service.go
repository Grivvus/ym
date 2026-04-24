package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserService struct {
	queries        *db.Queries
	st             storage.Storage
	logger         *slog.Logger
	artworkService ArtworkManager
}

func NewUserService(q *db.Queries, st storage.Storage, logger *slog.Logger) UserService {
	svc := UserService{
		queries: q,
		st:      st,
		logger:  logger,
	}

	svc.artworkService = NewArtworkManager(st, svc.loadArtworkOwner, logger)

	return svc
}

func (u *UserService) loadArtworkOwner(
	ctx context.Context, ownerID int32,
) (ArtworkOwner, error) {
	user, err := u.GetUserByID(ctx, ownerID)
	if err != nil {
		return ArtworkOwner{}, err
	}
	return ArtworkOwner{
		ID:   user.Id,
		Name: user.Username,
		Kind: "user",
	}, nil
}

func (u *UserService) GetUserByID(
	ctx context.Context, userID int32,
) (api.UserReturn, error) {
	var ret api.UserReturn

	user, err := u.queries.GetUserByID(ctx, userID)
	if err != nil {
		slog.Error("Get-userByID", "err", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("user", userID)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	ret.Username = user.Username
	ret.Id = user.ID
	ret.IsSuperuser = user.IsSuperuser
	if user.Email.Valid {
		ret.Email = &user.Email.String
	} else {
		ret.Email = nil
	}

	return ret, nil
}

func (u *UserService) ChangeUser(
	ctx context.Context,
	userID int32,
	newUserParams api.UserUpdate,
) (api.UserReturn, error) {
	updateParamsDB := db.UpdateUserParams{
		ID:       userID,
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
			return ret, NewErrNotFound("user", userID)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	ret.Username = updatedUser.Username
	ret.Id = updatedUser.ID
	if updatedUser.Email.Valid {
		ret.Email = &updatedUser.Email.String
	} else {
		ret.Email = nil
	}

	return ret, nil
}

func (u *UserService) ChangePassword(
	ctx context.Context, userID int32, newPasswordParams api.UserChangePassword,
) error {
	user, err := u.queries.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NewErrNotFound("user", userID)
		}
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	if !utils.VerifyPassword(newPasswordParams.OldPassword, user.Salt, user.Password) {
		return fmt.Errorf("%w: old password is wrong", ErrBadParams)
	}
	newHashed, newSalt := utils.HashPassword(newPasswordParams.NewPassword)
	err = u.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:       userID,
		Password: newHashed,
		Salt:     newSalt,
	})
	if err != nil {
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return nil
}

func (u *UserService) UploadAvatar(
	ctx context.Context, userID int32, avatar io.Reader,
) error {
	return u.artworkService.Upload(ctx, userID, avatar)
}

func (u *UserService) GetAvatar(ctx context.Context, userID int32) ([]byte, error) {
	return u.artworkService.Get(ctx, userID)
}

func (u *UserService) DeleteAvatar(
	ctx context.Context, userID int32,
) error {
	return u.artworkService.Delete(ctx, userID)
}
