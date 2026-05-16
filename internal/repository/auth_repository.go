package repository

import (
	"context"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository interface {
	CreateUserWithInitialRole(ctx context.Context, params CreateAuthUserParams) (AuthUser, error)
	GetUserByUsername(ctx context.Context, username string) (AuthUser, error)
	GetUserByID(ctx context.Context, userID int32) (AuthUser, error)
	UpdateUserPasswordHashParams(ctx context.Context, userID int32, params PasswordHashParams) error
}

type CreateAuthUserParams struct {
	Username           string
	Password           []byte
	Salt               []byte
	PasswordHashParams PasswordHashParams
}

type AuthUser struct {
	ID                 int32
	Username           string
	Password           []byte
	Salt               []byte
	PasswordHashParams PasswordHashParams
	IsSuperuser        bool
	RefreshVersion     int32
}

type PostgresAuthRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewAuthRepository(pool *pgxpool.Pool) *PostgresAuthRepository {
	return &PostgresAuthRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

func (repo *PostgresAuthRepository) CreateUserWithInitialRole(
	ctx context.Context, params CreateAuthUserParams,
) (AuthUser, error) {
	createdUser, err := withTx(ctx, repo.pool, repo.queries, func(q *db.Queries) (db.User, error) {
		usersCnt, err := q.GetUserCount(ctx)
		if err != nil {
			return db.User{}, err
		}

		createdUser, err := q.CreateUser(ctx, db.CreateUserParams{
			Username:            params.Username,
			Email:               pgtype.Text{Valid: false},
			Password:            params.Password,
			Salt:                params.Salt,
			PasswordMemory:      params.PasswordHashParams.Memory,
			PasswordIterations:  params.PasswordHashParams.Iterations,
			PasswordParallelism: params.PasswordHashParams.Parallelism,
			PasswordKeyLength:   params.PasswordHashParams.KeyLength,
			IsSuperuser:         usersCnt == 0,
		})
		if err != nil {
			return db.User{}, err
		}

		return createdUser, nil
	})
	if err != nil {
		return AuthUser{}, wrapDBError(err)
	}
	return authUserFromDBUser(createdUser), nil
}

func (repo *PostgresAuthRepository) GetUserByUsername(
	ctx context.Context, username string,
) (AuthUser, error) {
	user, err := repo.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return AuthUser{}, wrapDBError(err)
	}
	return AuthUser{
		ID:                 user.ID,
		Username:           user.Username,
		Password:           user.Password,
		Salt:               user.Salt,
		PasswordHashParams: passwordHashParamsFromUserByUsernameRow(user),
		IsSuperuser:        user.IsSuperuser,
		RefreshVersion:     user.RefreshVersion,
	}, nil
}

func (repo *PostgresAuthRepository) GetUserByID(
	ctx context.Context, userID int32,
) (AuthUser, error) {
	user, err := repo.queries.GetUserByID(ctx, userID)
	if err != nil {
		return AuthUser{}, wrapDBError(err)
	}
	return AuthUser{
		ID:                 user.ID,
		Username:           user.Username,
		Password:           user.Password,
		Salt:               user.Salt,
		PasswordHashParams: passwordHashParamsFromUserByIDRow(user),
		IsSuperuser:        user.IsSuperuser,
		RefreshVersion:     user.RefreshVersion,
	}, nil
}

func (repo *PostgresAuthRepository) UpdateUserPasswordHashParams(
	ctx context.Context, userID int32, params PasswordHashParams,
) error {
	err := repo.queries.UpdateUserPasswordHashParams(
		ctx,
		db.UpdateUserPasswordHashParamsParams{
			ID:                  userID,
			PasswordMemory:      params.Memory,
			PasswordIterations:  params.Iterations,
			PasswordParallelism: params.Parallelism,
			PasswordKeyLength:   params.KeyLength,
		},
	)
	return wrapDBError(err)
}

func authUserFromDBUser(user db.User) AuthUser {
	return AuthUser{
		ID:                 user.ID,
		Username:           user.Username,
		Password:           user.Password,
		Salt:               user.Salt,
		PasswordHashParams: passwordHashParamsFromDBUser(user),
		IsSuperuser:        user.IsSuperuser,
		RefreshVersion:     user.RefreshVersion,
	}
}
