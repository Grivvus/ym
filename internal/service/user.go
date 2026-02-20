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
	"github.com/Grivvus/ym/internal/transcoder"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserService struct {
	queries *db.Queries
	st      storage.Storage
}

func NewUserService(q *db.Queries, st storage.Storage) UserService {
	return UserService{
		queries: q,
		st:      st,
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
			return ret, NewErrNotFound("user", userID)
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
			return ret, NewErrNotFound("user", userID)
		}
		return ret, fmt.Errorf("unkown db error: %w", err)
	}

	ret.Username = updatedUser.Username
	if updatedUser.Email.Valid {
		ret.Email = &updatedUser.Email.String
	} else {
		ret.Email = nil
	}

	return ret, nil
}

func (u *UserService) ChangePassword(
	ctx context.Context, userID int, newPasswordParams api.UserChangePassword,
) error {
	user, err := u.queries.GetUserByID(ctx, int32(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NewErrNotFound("user", userID)
		}
		return fmt.Errorf("unkown db error: %w", err)
	}
	if !utils.VerifyPassword(newPasswordParams.OldPassword, user.Salt, user.Password) {
		return fmt.Errorf("wrong password")
	}
	newHashed, newSalt := utils.HashPassword(newPasswordParams.NewPassword)
	err = u.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:       int32(userID),
		Password: newHashed,
		Salt:     newSalt,
	})
	if err != nil {
		return fmt.Errorf("unkown db error: %w", err)
	}
	return nil
}

func (u *UserService) UploadAvatar(
	ctx context.Context, userID int, avatar io.Reader,
) error {
	rcTranscoded, err := transcoder.ToWebp(avatar)
	if err != nil {
		return fmt.Errorf("can't transcode image: %w", err)
	}
	defer func() { _ = rcTranscoded.Close() }()

	err = u.st.PutImage(ctx, ImageID("user", userID, ""), rcTranscoded)
	if err != nil {
		return fmt.Errorf("can't upload avatar: %w", err)
	}
	return nil
}

func (u *UserService) DeleteAvatar(
	ctx context.Context, userID int,
) error {
	err := u.st.RemoveImage(ctx, ImageID("user", userID, ""))
	if err != nil {
		// no switch on error, if i want to distinguish errors
		// i should create my own on a storage level, so they
		// will be indepentent from a concrete storage (i.e. minio, fs)
		return fmt.Errorf("storage error: %w", err)
	}
	return nil
}
