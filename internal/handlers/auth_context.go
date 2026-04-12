package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Grivvus/ym/internal/service"
)

type authenticatedUserIDKey struct{}

func withAuthenticatedUserID(ctx context.Context, userID int32) context.Context {
	return context.WithValue(ctx, authenticatedUserIDKey{}, userID)
}

func authenticatedUserID(ctx context.Context) (int32, bool) {
	userID, ok := ctx.Value(authenticatedUserIDKey{}).(int32)
	return userID, ok
}

func requireAuthenticatedUserID(w http.ResponseWriter, r *http.Request) (int32, bool) {
	userID, ok := authenticatedUserID(r.Context())
	if !ok {
		_ = WriteError(
			w, http.StatusUnauthorized,
			fmt.Errorf("authenticated user id not found"),
		)
		return 0, false
	}
	return userID, true
}

func requireCurrentUser(w http.ResponseWriter, r *http.Request, targetUserID int32) bool {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return false
	}
	if userID != targetUserID {
		_ = WriteError(
			w, http.StatusUnauthorized,
			fmt.Errorf("user is not allowed to access this resource"),
		)
		return false
	}
	return true
}

func requireSuperuser(
	w http.ResponseWriter, r *http.Request, auth service.AuthService,
) bool {
	user, authenticated := requireAuthenticatedUserID(w, r)
	if !authenticated {
		return false
	}
	if err := auth.AuthorizeSuperuser(r.Context(), user); err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			_ = WriteError(w, http.StatusUnauthorized, err)
			return false
		}
		if errors.Is(err, service.ErrSuperuserRequired) {
			_ = WriteError(w, http.StatusForbidden, err)
			return false
		}
	}
	return true
}
