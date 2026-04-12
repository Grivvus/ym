package handlers

import (
	"log/slog"
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
	BackupHandlers
}

func NewRootHandler(
	logger *slog.Logger,
	auth service.AuthService, user service.UserService,
	album service.AlbumService, artist service.ArtistService,
	track service.TrackService, playlist service.PlaylistService,
	backup service.BackupService,
) RootHandler {
	return RootHandler{
		AuthHandlers:     AuthHandlers{service: auth, logger: logger},
		ArtistHandlers:   ArtistHandlers{artistService: artist, logger: logger},
		UserHandlers:     UserHandlers{userService: user, logger: logger},
		AlbumHandlers:    AlbumHandlers{albumService: album, logger: logger},
		TrackHandlers:    TrackHandlers{trackService: track, logger: logger},
		PlaylistHandlers: PlaylistHandlers{playlistService: playlist, logger: logger},
		BackupHandlers: BackupHandlers{
			logger: logger, backupService: backup,
			authService: auth,
		},
	}
}

func (h RootHandler) Ping(w http.ResponseWriter, r *http.Request) {
	_ = r
	w.WriteHeader(http.StatusOK)
}
