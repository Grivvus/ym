package service

import (
	"testing"

	"github.com/Grivvus/ym/internal/audio"
	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindClosestExistingTrackKey(t *testing.T) {
	t.Parallel()

	text := func(s string) pgtype.Text {
		return pgtype.Text{String: s, Valid: true}
	}

	baseTrack := db.GetTrackRow{
		ID:                  42,
		FastPresetFname:     pgtype.Text{},
		StandardPresetFname: pgtype.Text{},
		HighPresetFname:     pgtype.Text{},
		LosslessPresetFname: pgtype.Text{},
	}

	testCases := []struct {
		name   string
		track  db.GetTrackRow
		preset audio.Preset
		want   string
	}{
		{
			name: "lossless exact match",
			track: db.GetTrackRow{
				ID:                  42,
				LosslessPresetFname: text("track42_lossless"),
			},
			preset: audio.PresetLossless,
			want:   "track42_lossless",
		},
		{
			name:   "lossless falls back to original",
			track:  baseTrack,
			preset: audio.PresetLossless,
			want:   "track42",
		},
		{
			name: "high falls back to standard",
			track: db.GetTrackRow{
				ID:                  42,
				StandardPresetFname: text("track42_standard"),
				FastPresetFname:     text("track42_fast"),
			},
			preset: audio.PresetHigh,
			want:   "track42_standard",
		},
		{
			name: "high falls back to fast",
			track: db.GetTrackRow{
				ID:              42,
				FastPresetFname: text("track42_fast"),
			},
			preset: audio.PresetHigh,
			want:   "track42_fast",
		},
		{
			name:   "high falls back to original",
			track:  baseTrack,
			preset: audio.PresetHigh,
			want:   "track42",
		},
		{
			name: "standard falls back to fast",
			track: db.GetTrackRow{
				ID:              42,
				FastPresetFname: text("track42_fast"),
			},
			preset: audio.PresetStandard,
			want:   "track42_fast",
		},
		{
			name: "standard falls back to high",
			track: db.GetTrackRow{
				ID:              42,
				HighPresetFname: text("track42_high"),
			},
			preset: audio.PresetStandard,
			want:   "track42_high",
		},
		{
			name:   "standard falls back to original",
			track:  baseTrack,
			preset: audio.PresetStandard,
			want:   "track42",
		},
		{
			name: "fast falls back to standard",
			track: db.GetTrackRow{
				ID:                  42,
				StandardPresetFname: text("track42_standard"),
			},
			preset: audio.PresetFast,
			want:   "track42_standard",
		},
		{
			name: "fast falls back to high",
			track: db.GetTrackRow{
				ID:              42,
				HighPresetFname: text("track42_high"),
			},
			preset: audio.PresetFast,
			want:   "track42_high",
		},
		{
			name:   "fast falls back to original",
			track:  baseTrack,
			preset: audio.PresetFast,
			want:   "track42",
		},
		{
			name:   "invalid preset returns error",
			track:  baseTrack,
			preset: audio.Preset(0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := findClosestExistingTrackKey(tc.track, tc.preset)
			if tc.want == "" {
				require.Error(t, err, "expected error, got nil")
				assert.ErrorIs(t, err, ErrPresetCantBeSelected)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFindClosestExistingTrackFileResolvedQuality(t *testing.T) {
	t.Parallel()

	text := func(s string) pgtype.Text {
		return pgtype.Text{String: s, Valid: true}
	}

	testCases := []struct {
		name        string
		track       db.GetTrackRow
		preset      audio.Preset
		wantKey     string
		wantQuality string
	}{
		{
			name: "standard falls back to fast",
			track: db.GetTrackRow{
				ID:              42,
				FastPresetFname: text("track42_fast"),
			},
			preset:      audio.PresetStandard,
			wantKey:     "track42_fast",
			wantQuality: "fast",
		},
		{
			name: "high exact match",
			track: db.GetTrackRow{
				ID:              42,
				HighPresetFname: text("track42_high"),
			},
			preset:      audio.PresetHigh,
			wantKey:     "track42_high",
			wantQuality: "high",
		},
		{
			name:        "lossless falls back to original",
			track:       db.GetTrackRow{ID: 42},
			preset:      audio.PresetLossless,
			wantKey:     "track42",
			wantQuality: "original",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := findClosestExistingTrackFile(tc.track, tc.preset)
			require.NoError(t, err)
			assert.Equal(t, tc.wantKey, got.key)
			assert.Equal(t, tc.wantQuality, got.quality)
		})
	}
}

func TestCanAccessTrack(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		track  db.GetTrackRow
		userID int32
		want   bool
	}{
		{
			name:   "globally available track is accessible",
			track:  db.GetTrackRow{IsGloballyAvailable: true},
			userID: 10,
			want:   true,
		},
		{
			name: "uploaded private track is accessible to uploader",
			track: db.GetTrackRow{
				UploadByUser: pgtype.Int4{Int32: 10, Valid: true},
			},
			userID: 10,
			want:   true,
		},
		{
			name: "private track is forbidden for another user",
			track: db.GetTrackRow{
				UploadByUser: pgtype.Int4{Int32: 10, Valid: true},
			},
			userID: 11,
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, canAccessTrack(tc.track, tc.userID))
		})
	}
}

func TestTrackDownloadFileName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "track-42-standard.opus", trackDownloadFileName(42, "standard", "audio/ogg"))
	assert.Equal(t, "track-42-high.m4a", trackDownloadFileName(42, "high", "audio/mp4"))
	assert.Equal(t, "track-42-original.mp3", trackDownloadFileName(42, "original", "audio/mpeg"))
}
