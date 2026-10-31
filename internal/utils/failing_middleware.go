package utils

import (
	"log/slog"
	"math/rand/v2"
	"net/http"
)

func FailingMiddleware(logger *slog.Logger, failRate float64) func(http.Handler) http.Handler {
	if failRate > 1.0 {
		failRate = 1.0
	}
	if failRate < 0 {
		failRate = 0
	}
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if rand.Float64() < failRate {
				logger.Debug("request intentionally failed with 504")
				w.WriteHeader(http.StatusGatewayTimeout)
				return
			}
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
