package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserService struct {
	queries        *db.Queries
	objStorage     storage.Storage
	logger         *slog.Logger
	artworkService ArtworkManager
}

func NewUserService(q *db.Queries, st storage.Storage, logger *slog.Logger) UserService {
	svc := UserService{
		queries:    q,
		objStorage: st,
		logger:     logger,
	}

	svc.artworkService = NewArtworkManager(st, svc.loadArtworkOwner, logger)

	return svc
}

func (u *UserService) loadArtworkOwner(
	ctx context.Context, ownerID int32,
) (ArtworkOwner, error) {
	user, err := u.GetUser(ctx, ownerID)
	if err != nil {
		return ArtworkOwner{}, err
	}
	return ArtworkOwner{
		ID:   user.Id,
		Name: user.Username,
		Kind: "user",
	}, nil
}

func (u *UserService) GetAllUsers(ctx context.Context) (api.Users, error) {
	users, err := u.queries.GetAllUsernames(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w, caused by - %w", ErrUnknownDBError, err)
	}
	apiUsers := make(api.Users, len(users))
	for i, user := range users {
		apiUsers[i] = api.SimpleUser{
			Id:       user.ID,
			Username: user.Username,
		}
	}
	return apiUsers, nil
}

func (u *UserService) GetUser(
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
	currentUser, err := u.queries.GetUserByID(ctx, userID)
	var ret api.UserReturn
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("user", userID)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	username := currentUser.Username
	if newUserParams.NewUsername != nil {
		username = strings.TrimSpace(*newUserParams.NewUsername)
		if username == "" {
			return ret, fmt.Errorf("%w: new username is required", ErrBadParams)
		}
	}

	email := currentUser.Email
	if newUserParams.NewEmail != nil {
		if strings.TrimSpace(string(*newUserParams.NewEmail)) == "" {
			email = pgtype.Text{Valid: false}
		} else {
			normalizedEmail, err := utils.NormalizeEmailAddress(string(*newUserParams.NewEmail))
			if err != nil {
				return ret, fmt.Errorf("%w: %w", ErrBadParams, err)
			}
			email = pgtype.Text{
				String: normalizedEmail,
				Valid:  true,
			}
		}
	}

	updateParamsDB := db.UpdateUserParams{
		ID:       userID,
		Username: username,
		Email:    email,
	}

	updatedUser, err := u.queries.UpdateUser(ctx, updateParamsDB)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("user", userID)
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return ret, NewErrAlreadyExists("user", username)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	ret.Username = updatedUser.Username
	ret.Id = updatedUser.ID
	ret.IsSuperuser = updatedUser.IsSuperuser
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
	if strings.TrimSpace(newPasswordParams.NewPassword) == "" {
		return fmt.Errorf("%w: new password is required", ErrBadParams)
	}
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
