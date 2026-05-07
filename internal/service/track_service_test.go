package service

import (
	"testing"

	"github.com/Grivvus/ym/internal/audio"
	"github.com/Grivvus/ym/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindClosestExistingTrackKey(t *testing.T) {
	t.Parallel()

	text := func(s string) *string {
		return &s
	}

	baseTrack := repository.Track{
		ID: 42,
	}

	testCases := []struct {
		name   string
		track  repository.Track
		preset audio.Preset
		want   string
	}{
		{
			name: "lossless exact match",
			track: repository.Track{
				ID:                 42,
				LosslessPresetName: text("track42_lossless"),
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
			track: repository.Track{
				ID:                 42,
				StandardPresetName: text("track42_standard"),
				FastPresetName:     text("track42_fast"),
			},
			preset: audio.PresetHigh,
			want:   "track42_standard",
		},
		{
			name: "high falls back to fast",
			track: repository.Track{
				ID:             42,
				FastPresetName: text("track42_fast"),
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
			track: repository.Track{
				ID:             42,
				FastPresetName: text("track42_fast"),
			},
			preset: audio.PresetStandard,
			want:   "track42_fast",
		},
		{
			name: "standard falls back to high",
			track: repository.Track{
				ID:             42,
				HighPresetName: text("track42_high"),
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
			track: repository.Track{
				ID:                 42,
				StandardPresetName: text("track42_standard"),
			},
			preset: audio.PresetFast,
			want:   "track42_standard",
		},
		{
			name: "fast falls back to high",
			track: repository.Track{
				ID:             42,
				HighPresetName: text("track42_high"),
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

	text := func(s string) *string {
		return &s
	}

	testCases := []struct {
		name        string
		track       repository.Track
		preset      audio.Preset
		wantKey     string
		wantQuality string
	}{
		{
			name: "standard falls back to fast",
			track: repository.Track{
				ID:             42,
				FastPresetName: text("track42_fast"),
			},
			preset:      audio.PresetStandard,
			wantKey:     "track42_fast",
			wantQuality: "fast",
		},
		{
			name: "high exact match",
			track: repository.Track{
				ID:             42,
				HighPresetName: text("track42_high"),
			},
			preset:      audio.PresetHigh,
			wantKey:     "track42_high",
			wantQuality: "high",
		},
		{
			name:        "lossless falls back to original",
			track:       repository.Track{ID: 42},
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
		track  repository.Track
		userID int32
		want   bool
	}{
		{
			name:   "globally available track is accessible",
			track:  repository.Track{IsGloballyAvailable: true},
			userID: 10,
			want:   true,
		},
		{
			name: "uploaded private track is accessible to uploader",
			track: repository.Track{
				UploadBy: ptr(int32(10)),
			},
			userID: 10,
			want:   true,
		},
		{
			name: "private track is forbidden for another user",
			track: repository.Track{
				UploadBy: ptr(int32(10)),
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

func ptr[T any](value T) *T {
	return &value
}
