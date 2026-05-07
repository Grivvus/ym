package repository

import (
	"context"
	"time"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PasswordResetRepository interface {
	GetUserByEmail(ctx context.Context, email string) (PasswordResetUser, error)
	GetPasswordResetCode(ctx context.Context, userID int32) (PasswordResetCode, error)
	UpsertPasswordResetCode(ctx context.Context, params UpsertPasswordResetCodeParams) error
	DeletePasswordResetCode(ctx context.Context, userID int32) error
	UpdatePasswordResetCodeAttempts(ctx context.Context, userID, attemptsLeft int32) error
	UpdateUserPassword(ctx context.Context, params PasswordResetUpdatePasswordParams) error
}

type PasswordResetUser struct {
	ID int32
}

type PasswordResetCode struct {
	UserID            int32
	CodeHash          []byte
	ExpiresAt         *time.Time
	AttemptsLeft      int32
	ResendAvailableAt *time.Time
}

type UpsertPasswordResetCodeParams struct {
	UserID            int32
	CodeHash          []byte
	ExpiresAt         time.Time
	AttemptsLeft      int32
	ResendAvailableAt time.Time
}

type PasswordResetUpdatePasswordParams struct {
	UserID   int32
	Password []byte
	Salt     []byte
}

type PostgresPasswordResetRepository struct {
	queries *db.Queries
}

func NewPasswordResetRepository(pool *pgxpool.Pool) *PostgresPasswordResetRepository {
	return &PostgresPasswordResetRepository{
		queries: db.New(pool),
	}
}

func (repo *PostgresPasswordResetRepository) GetUserByEmail(
	ctx context.Context, email string,
) (PasswordResetUser, error) {
	user, err := repo.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return PasswordResetUser{}, wrapDBError(err)
	}
	return PasswordResetUser{
		ID: user.ID,
	}, nil
}

func (repo *PostgresPasswordResetRepository) GetPasswordResetCode(
	ctx context.Context, userID int32,
) (PasswordResetCode, error) {
	code, err := repo.queries.GetPasswordResetCodeByUserID(ctx, userID)
	if err != nil {
		return PasswordResetCode{}, wrapDBError(err)
	}
	return PasswordResetCode{
		UserID:            code.UserID,
		CodeHash:          code.CodeHash,
		ExpiresAt:         timePtrFromPGTimestamptz(code.ExpiresAt),
		AttemptsLeft:      code.AttemptsLeft,
		ResendAvailableAt: timePtrFromPGTimestamptz(code.ResendAvailableAt),
	}, nil
}

func (repo *PostgresPasswordResetRepository) UpsertPasswordResetCode(
	ctx context.Context, params UpsertPasswordResetCodeParams,
) error {
	err := repo.queries.UpsertPasswordResetCode(ctx, db.UpsertPasswordResetCodeParams{
		UserID:            params.UserID,
		CodeHash:          params.CodeHash,
		ExpiresAt:         pgTimestamptzFromTime(params.ExpiresAt),
		AttemptsLeft:      params.AttemptsLeft,
		ResendAvailableAt: pgTimestamptzFromTime(params.ResendAvailableAt),
	})
	return wrapDBError(err)
}

func (repo *PostgresPasswordResetRepository) DeletePasswordResetCode(
	ctx context.Context, userID int32,
) error {
	err := repo.queries.DeletePasswordResetCodeByUserID(ctx, userID)
	return wrapDBError(err)
}

func (repo *PostgresPasswordResetRepository) UpdatePasswordResetCodeAttempts(
	ctx context.Context, userID, attemptsLeft int32,
) error {
	err := repo.queries.UpdatePasswordResetCodeAttempts(
		ctx,
		db.UpdatePasswordResetCodeAttemptsParams{
			UserID:       userID,
			AttemptsLeft: attemptsLeft,
		},
	)
	return wrapDBError(err)
}

func (repo *PostgresPasswordResetRepository) UpdateUserPassword(
	ctx context.Context, params PasswordResetUpdatePasswordParams,
) error {
	err := repo.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:       params.UserID,
		Password: params.Password,
		Salt:     params.Salt,
	})
	return wrapDBError(err)
}

func pgTimestamptzFromTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value,
		Valid: true,
	}
}

func timePtrFromPGTimestamptz(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
