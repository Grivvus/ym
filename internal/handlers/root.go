package handlers

import "github.com/Grivvus/ym/internal/service"

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
		UserHandlers:     UserHandlers{user},
		AlbumHandlers:    AlbumHandlers{album},
		TrackHandlers:    TrackHandlers{track},
		PlaylistHandlers: PlaylistHandlers{playlist},
	}
}
