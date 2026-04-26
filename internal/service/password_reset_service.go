package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/mailer"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const defaultPasswordResetAcceptedMessage = "if an account with that email exists, a reset code has been sent"

type PasswordResetService struct {
	queries         *db.Queries
	logger          *slog.Logger
	mailer          mailer.Mailer
	available       bool
	codeSecret      []byte
	codeTTL         time.Duration
	resendCooldown  time.Duration
	maxAttempts     int32
	codeLength      int
	acceptedMessage string
}

func NewPasswordResetService(
	q *db.Queries,
	logger *slog.Logger,
	m mailer.Mailer,
	cfg *utils.PasswordResetConfig,
) PasswordResetService {
	service := PasswordResetService{
		queries:         q,
		logger:          logger,
		mailer:          m,
		acceptedMessage: defaultPasswordResetAcceptedMessage,
	}

	if cfg == nil {
		return service
	}

	service.codeSecret = []byte(cfg.CodeSecret)
	service.codeTTL = cfg.CodeTTL
	service.resendCooldown = cfg.ResendCooldown
	service.maxAttempts = int32(cfg.MaxAttempts)
	service.codeLength = cfg.CodeLength
	if cfg.AcceptedMessage != "" {
		service.acceptedMessage = cfg.AcceptedMessage
	}
	service.available = cfg.Enabled && m != nil && cfg.Validate() == nil

	return service
}

func (s PasswordResetService) AcceptedMessage() string {
	return s.acceptedMessage
}

func (s PasswordResetService) RequestPasswordReset(
	ctx context.Context, req api.PasswordResetRequest,
) error {
	if !s.available {
		return fmt.Errorf("%w: password reset is not available", ErrServiceUnavailable)
	}
	s.logger.Info("password reset operation initiated")

	email, err := utils.NormalizeEmailAddress(string(req.Email))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrBadParams, err)
	}

	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Info("no user found for email", "email", email)
			return nil
		}
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	now := time.Now()
	resetCode, err := s.queries.GetPasswordResetCodeByUserID(ctx, user.ID)
	switch {
	case err == nil:
		if resetCode.AttemptsLeft > 0 && resetCode.ExpiresAt.Valid && resetCode.ExpiresAt.Time.After(now) {
			if resetCode.ResendAvailableAt.Valid && resetCode.ResendAvailableAt.Time.After(now) {
				s.logger.Info("resend is not available yet")
				return nil
			}
		}
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	code, err := generatePasswordResetCode(s.codeLength)
	if err != nil {
		return fmt.Errorf("generate password reset code: %w", err)
	}

	err = s.queries.UpsertPasswordResetCode(ctx, db.UpsertPasswordResetCodeParams{
		UserID:            user.ID,
		CodeHash:          hashPasswordResetCode(s.codeSecret, code),
		ExpiresAt:         toPGTimestamptz(now.Add(s.codeTTL)),
		AttemptsLeft:      s.maxAttempts,
		ResendAvailableAt: toPGTimestamptz(now.Add(s.resendCooldown)),
	})
	if err != nil {
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	if err := s.mailer.SendPasswordResetCode(ctx, email, code, s.codeTTL); err != nil {
		s.logger.Error("failed to send password reset email", "email", email, "err", err)
		if deleteErr := s.queries.DeletePasswordResetCodeByUserID(ctx, user.ID); deleteErr != nil {
			s.logger.Error(
				"failed to delete unsent password reset code",
				"user_id", user.ID,
				"err", deleteErr,
			)
		}
	}

	return nil
}

func (s PasswordResetService) ConfirmPasswordReset(
	ctx context.Context, req api.PasswordResetConfirmRequest,
) error {
	if !s.available {
		return fmt.Errorf("%w: password reset is not available", ErrServiceUnavailable)
	}

	email, err := utils.NormalizeEmailAddress(string(req.Email))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrBadParams, err)
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		return invalidPasswordResetCodeError()
	}
	if strings.TrimSpace(req.NewPassword) == "" {
		return fmt.Errorf("%w: new password is required", ErrBadParams)
	}

	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return invalidPasswordResetCodeError()
		}
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	resetCode, err := s.queries.GetPasswordResetCodeByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return invalidPasswordResetCodeError()
		}
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	now := time.Now()
	if resetCode.AttemptsLeft <= 0 || !resetCode.ExpiresAt.Valid || !resetCode.ExpiresAt.Time.After(now) {
		_ = s.queries.DeletePasswordResetCodeByUserID(ctx, user.ID)
		return invalidPasswordResetCodeError()
	}

	expectedHash := hashPasswordResetCode(s.codeSecret, code)
	if subtle.ConstantTimeCompare(expectedHash, resetCode.CodeHash) != 1 {
		if resetCode.AttemptsLeft <= 1 {
			_ = s.queries.DeletePasswordResetCodeByUserID(ctx, user.ID)
		} else {
			err := s.queries.UpdatePasswordResetCodeAttempts(
				ctx,
				db.UpdatePasswordResetCodeAttemptsParams{
					UserID:       user.ID,
					AttemptsLeft: resetCode.AttemptsLeft - 1,
				},
			)
			if err != nil {
				return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
			}
		}
		return invalidPasswordResetCodeError()
	}

	hashedPassword, salt := utils.HashPassword(req.NewPassword)
	err = s.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:       user.ID,
		Password: hashedPassword,
		Salt:     salt,
	})
	if err != nil {
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	return nil
}

func invalidPasswordResetCodeError() error {
	return fmt.Errorf("%w: invalid or expired reset code", ErrBadParams)
}

func generatePasswordResetCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("password reset code length must be positive")
	}

	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	code := make([]byte, length)
	for i, value := range bytes {
		code[i] = '0' + (value % 10)
	}

	return string(code), nil
}

func hashPasswordResetCode(secret []byte, code string) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(code))
	return mac.Sum(nil)
}

func toPGTimestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value,
		Valid: true,
	}
}
