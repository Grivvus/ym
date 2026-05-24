package transcoder

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToWebpWithOptionsResizesArtwork(t *testing.T) {
	t.Parallel()

	src := testPNG(t, 2400, 1200)

	rc, err := ToWebpWithOptions(bytes.NewReader(src), ArtworkImageOptions())
	require.NoError(t, err)

	cfg := decodeWebPConfig(t, rc)
	assert.Equal(t, 1200, cfg.Width)
	assert.Equal(t, 600, cfg.Height)
}

func TestToWebpWithOptionsCropsAvatarToSquare(t *testing.T) {
	t.Parallel()

	src := testPNG(t, 1600, 800)

	rc, err := ToWebpWithOptions(bytes.NewReader(src), AvatarImageOptions())
	require.NoError(t, err)

	cfg := decodeWebPConfig(t, rc)
	assert.Equal(t, 512, cfg.Width)
	assert.Equal(t, 512, cfg.Height)
}

func TestToWebpWithOptionsRejectsTooManyPixels(t *testing.T) {
	t.Parallel()

	src := testPNG(t, 2, 2)
	opts := ArtworkImageOptions()
	opts.MaxInputPixels = 3

	_, err := ToWebpWithOptions(bytes.NewReader(src), opts)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrImageTooLarge))
}

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func decodeWebPConfig(t *testing.T, rc io.ReadCloser) image.Config {
	t.Helper()
	defer func() { require.NoError(t, rc.Close()) }()

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, "webp", format)
	return cfg
}
