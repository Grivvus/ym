package handlers

import "github.com/Grivvus/ym/internal/service"

type RootHandler struct {
	AuthHandlers
	UserHandlers
}

func NewRootHandler(
	auth service.AuthService, user service.UserService,
) RootHandler {
	return RootHandler{
		AuthHandlers: AuthHandlers{auth},
		UserHandlers: UserHandlers{user},
	}
}
