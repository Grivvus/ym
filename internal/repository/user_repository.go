package repository

import (
	"context"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	GetAllUsers(ctx context.Context) ([]UserSummary, error)
	GetUserByID(ctx context.Context, userID int32) (User, error)
	UpdateUser(ctx context.Context, params UpdateUserParams) (User, error)
	UpdateUserPassword(ctx context.Context, params UpdateUserPasswordParams) error
}

type UserSummary struct {
	ID       int32
	Username string
}

type User struct {
	ID                 int32
	Username           string
	Email              *string
	Password           []byte
	Salt               []byte
	PasswordHashParams PasswordHashParams
	IsSuperuser        bool
	RefreshVersion     int32
}

type UpdateUserParams struct {
	ID       int32
	Username string
	Email    *string
}

type UpdateUserPasswordParams struct {
	ID                 int32
	Password           []byte
	Salt               []byte
	PasswordHashParams PasswordHashParams
}

type PostgresUserRepository struct {
	queries *db.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{
		queries: db.New(pool),
	}
}

func (repo *PostgresUserRepository) GetAllUsers(ctx context.Context) ([]UserSummary, error) {
	users, err := repo.queries.GetAllUsernames(ctx)
	if err != nil {
		return nil, wrapDBError(err)
	}

	result := make([]UserSummary, len(users))
	for i, user := range users {
		result[i] = UserSummary{
			ID:       user.ID,
			Username: user.Username,
		}
	}
	return result, nil
}

func (repo *PostgresUserRepository) GetUserByID(ctx context.Context, userID int32) (User, error) {
	user, err := repo.queries.GetUserByID(ctx, userID)
	if err != nil {
		return User{}, wrapDBError(err)
	}
	return userFromGetUserByIDRow(user), nil
}

func (repo *PostgresUserRepository) UpdateUser(
	ctx context.Context, params UpdateUserParams,
) (User, error) {
	updatedUser, err := repo.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:       params.ID,
		Username: params.Username,
		Email:    pgTextFromStringPtr(params.Email),
	})
	if err != nil {
		return User{}, wrapDBError(err)
	}
	return userFromDBUser(updatedUser), nil
}

func (repo *PostgresUserRepository) UpdateUserPassword(
	ctx context.Context, params UpdateUserPasswordParams,
) error {
	err := repo.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:                  params.ID,
		Password:            params.Password,
		Salt:                params.Salt,
		PasswordMemory:      params.PasswordHashParams.Memory,
		PasswordIterations:  params.PasswordHashParams.Iterations,
		PasswordParallelism: params.PasswordHashParams.Parallelism,
		PasswordKeyLength:   params.PasswordHashParams.KeyLength,
	})
	return wrapDBError(err)
}

func userFromGetUserByIDRow(user db.GetUserByIDRow) User {
	return User{
		ID:                 user.ID,
		Username:           user.Username,
		Email:              stringPtrFromPGText(user.Email),
		Password:           user.Password,
		Salt:               user.Salt,
		PasswordHashParams: passwordHashParamsFromUserByIDRow(user),
		IsSuperuser:        user.IsSuperuser,
		RefreshVersion:     user.RefreshVersion,
	}
}

func userFromDBUser(user db.User) User {
	return User{
		ID:                 user.ID,
		Username:           user.Username,
		Email:              stringPtrFromPGText(user.Email),
		Password:           user.Password,
		Salt:               user.Salt,
		PasswordHashParams: passwordHashParamsFromDBUser(user),
		IsSuperuser:        user.IsSuperuser,
		RefreshVersion:     user.RefreshVersion,
	}
}

func stringPtrFromPGText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func pgTextFromStringPtr(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *value, Valid: true}
}
