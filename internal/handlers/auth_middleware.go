package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Grivvus/ym/internal/utils"
)

func AuthMiddleware(logger *slog.Logger, jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublicPath(r.URL.Path) || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			token, err := bearerTokenFromHeader(r.Header.Get("Authorization"))
			if err != nil {
				_ = WriteError(w, http.StatusUnauthorized, err)
				return
			}

			userID, err := utils.ParseAccessToken(token, jwtSecret)
			if err != nil {
				logger.Warn("access token validation failed", "err", err)
				_ = WriteError(
					w, http.StatusUnauthorized,
					fmt.Errorf("invalid access token"),
				)
				return
			}

			next.ServeHTTP(w, r.WithContext(withAuthenticatedUserID(r.Context(), userID)))
		})
	}
}

func isPublicPath(path string) bool {
	switch path {
	case "/ping",
		"/auth/login",
		"/auth/register",
		"/auth/refresh",
		"/auth/password-reset/request",
		"/auth/password-reset/confirm",
		"/openapi.yml":
		return true
	default:
		return strings.HasPrefix(path, "/swagger/")
	}
}

func bearerTokenFromHeader(header string) (string, error) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("missing bearer token")
	}
	return strings.TrimSpace(token), nil
}
