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
	SearchHandler
}

func NewRootHandler(
	logger *slog.Logger,
	auth service.AuthService,
	passwordReset service.PasswordResetService,
	user service.UserService,
	album service.AlbumService, artist service.ArtistService,
	track service.TrackService, playlist service.PlaylistService,
	search service.SearchService,
	backup service.BackupService,
) RootHandler {
	return RootHandler{
		AuthHandlers: AuthHandlers{
			authService:          auth,
			passwordResetService: passwordReset,
			logger:               logger,
		},
		ArtistHandlers: ArtistHandlers{
			artistService: artist,
			authService:   auth,
			logger:        logger,
		},
		UserHandlers: UserHandlers{userService: user, logger: logger},
		AlbumHandlers: AlbumHandlers{
			albumService: album,
			authService:  auth,
			logger:       logger,
		},
		TrackHandlers:    TrackHandlers{trackService: track, logger: logger},
		PlaylistHandlers: PlaylistHandlers{playlistService: playlist, logger: logger},
		BackupHandlers: BackupHandlers{
			logger: logger, backupService: backup,
			authService: auth,
		},
		SearchHandler: SearchHandler{
			logger:  logger,
			service: search,
		},
	}
}

func (h RootHandler) Ping(w http.ResponseWriter, r *http.Request) {
	_ = r
	w.WriteHeader(http.StatusOK)
}
