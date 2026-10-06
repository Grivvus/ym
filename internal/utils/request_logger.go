package utils

import (
	"log/slog"
	"net/http"
)

func LoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			logger.Debug("request handle started", "uri", r.RequestURI, "method", r.Method)
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
