package dto

import "time"

type Album struct {
	ID       int32  `json:"id"`
	Name     string `json:"name"`
	ArtistID int32  `json:"artist_id"`
}

type Artist struct {
	ID   int32   `json:"id"`
	Name string  `json:"name"`
	URL  *string `json:"url"`
}

type Playlist struct {
	ID       int32  `json:"id"`
	Name     string `json:"name"`
	IsPublic bool   `json:"is_public"`
	OwnerID  *int32 `json:"owner_id"`
}

type Track struct {
	ID                  int32   `json:"id"`
	ArtistID            int32   `json:"artist_id"`
	Name                string  `json:"name"`
	DurationMs          *int32  `json:"duration_ms"`
	URL                 *string `json:"url"`
	FastPresetFname     *string `json:"fast_preset_fname"`
	StandardPresetFname *string `json:"standard_preset_fname"`
	HighPresetFname     *string `json:"high_preset_fname"`
	LosslessPresetFname *string `json:"lossless_preset_fname"`
	IsGloballyAvailable bool    `json:"is_globally_available"`
	UploadByUser        *int32  `json:"upload_by_user"`
}

type TrackAlbum struct {
	TrackID int32 `json:"track_id"`
	AlbumID int32 `json:"album_id"`
}

type TrackPlaylist struct {
	TrackID    int32 `json:"track_id"`
	PlaylistID int32 `json:"playlist_id"`
}

type User struct {
	ID          int32     `json:"id"`
	Username    string    `json:"username"`
	IsSuperuser bool      `json:"is_superuser"`
	Email       *string   `json:"email"`
	Password    string    `json:"password"`
	Salt        string    `json:"salt"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TranscodingQueueRow struct {
	ID                    int32     `json:"id"`
	TrackID               int32     `json:"track_id"`
	AddedAt               time.Time `json:"added_at"`
	TrackOriginalFileName string    `json:"track_original_file_name"`
	WasFailed             bool      `json:"was_failed"`
	ErrorMsg              *string   `json:"error_msg"`
}

type FullDBBackup struct {
	Users            []User                `json:"users"`
	Artists          []Artist              `json:"artists"`
	Albums           []Album               `json:"albums"`
	Playlists        []Playlist            `json:"playlists"`
	Tracks           []Track               `json:"tracks"`
	TrackAlbums      []TrackAlbum          `json:"track_albums"`
	TrackPlaylists   []TrackPlaylist       `json:"track_playlists"`
	TranscodingQueue []TranscodingQueueRow `json:"transcoding_queue"`
}
