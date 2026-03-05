package handlers

import (
	"net/http"

	"github.com/Grivvus/ym/internal/service"
)

type RootHandler struct {
	AuthHandlers
	UserHandlers
	AlbumHandlers
	ArtistHandlers
	TrackHandlers
	PlaylistHandlers
}

func NewRootHandler(
	auth service.AuthService, user service.UserService,
	album service.AlbumService, artist service.ArtistService,
	track service.TrackService, playlist service.PlaylistService,
) RootHandler {
	return RootHandler{
		AuthHandlers:     AuthHandlers{auth},
		ArtistHandlers:   ArtistHandlers{artist},
		UserHandlers:     UserHandlers{user},
		AlbumHandlers:    AlbumHandlers{album},
		TrackHandlers:    TrackHandlers{track},
		PlaylistHandlers: PlaylistHandlers{playlist},
	}
}

func (h RootHandler) Ping(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
