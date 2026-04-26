package mailer

import (
	"context"
	"time"
)

type Mailer interface {
	SendPasswordResetCode(
		ctx context.Context, recipient string, code string, ttl time.Duration,
	) error
}
