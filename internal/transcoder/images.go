package transcoder

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	xdraw "golang.org/x/image/draw"
)

var ErrCantDecode = errors.New("can't decode media")

var ErrCantEncode = errors.New("can't encode media")

var ErrImageTooLarge = errors.New("image is too large")

const (
	defaultMaxEncodedBytes   = 20 << 20
	defaultMaxInputDimension = 10000
	defaultMaxInputPixels    = 25_000_000
	defaultWebPQuality       = 82
	defaultWebPMethod        = 4
)

type ImageOptions struct {
	MaxWidth          int
	MaxHeight         int
	CropSquare        bool
	Quality           float32
	Method            int
	Preset            encoder.EncodingPreset
	MaxEncodedBytes   int64
	MaxInputDimension int
	MaxInputPixels    int
}

func ArtworkImageOptions() ImageOptions {
	return ImageOptions{
		MaxWidth:  1200,
		MaxHeight: 1200,
		Quality:   82,
		Method:    4,
		Preset:    encoder.PresetPicture,
	}
}

func AvatarImageOptions() ImageOptions {
	return ImageOptions{
		MaxWidth:   512,
		MaxHeight:  512,
		CropSquare: true,
		Quality:    80,
		Method:     4,
		Preset:     encoder.PresetPicture,
	}
}

func ToWebp(r io.Reader) (io.ReadCloser, error) {
	return ToWebpWithOptions(r, ArtworkImageOptions())
}

func ToWebpWithOptions(r io.Reader, opts ImageOptions) (io.ReadCloser, error) {
	opts = opts.withDefaults()
	data, err := readImagePayload(r, opts.MaxEncodedBytes)
	if err != nil {
		return nil, err
	}

	cfg, formatFound, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCantDecode, err)
	}
	if err := validateImageConfig(cfg, opts); err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCantDecode, err)
	}
	img = transformImage(img, opts)

	slog.Info(
		"image format detected",
		"format", formatFound,
		"input_width", cfg.Width,
		"input_height", cfg.Height,
		"output_width", img.Bounds().Dx(),
		"output_height", img.Bounds().Dy(),
	)
	pr, pw := io.Pipe()

	encoderOpts, err := encoder.NewLossyEncoderOptions(opts.Preset, opts.Quality)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCantEncode, err)
	}
	encoderOpts.Method = opts.Method
	encoderOpts.ThreadLevel = true
	encoderOpts.UseSharpYuv = true

	go func() {
		defer func() { _ = pw.Close() }()
		if err := webp.Encode(pw, img, encoderOpts); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("%w: %w", ErrCantEncode, err))
			return
		}
	}()

	return pr, nil
}

func (opts ImageOptions) withDefaults() ImageOptions {
	if opts.MaxEncodedBytes <= 0 {
		opts.MaxEncodedBytes = defaultMaxEncodedBytes
	}
	if opts.MaxInputDimension <= 0 {
		opts.MaxInputDimension = defaultMaxInputDimension
	}
	if opts.MaxInputPixels <= 0 {
		opts.MaxInputPixels = defaultMaxInputPixels
	}
	if opts.Quality <= 0 {
		opts.Quality = defaultWebPQuality
	}
	if opts.Method == 0 {
		opts.Method = defaultWebPMethod
	}
	return opts
}

func readImagePayload(r io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCantDecode, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%w: encoded image exceeds %d bytes", ErrImageTooLarge, maxBytes)
	}
	return data, nil
}

func validateImageConfig(cfg image.Config, opts ImageOptions) error {
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return fmt.Errorf("%w: invalid image dimensions", ErrCantDecode)
	}
	if cfg.Width > opts.MaxInputDimension || cfg.Height > opts.MaxInputDimension {
		return fmt.Errorf(
			"%w: dimensions %dx%d exceed %dpx",
			ErrImageTooLarge, cfg.Width, cfg.Height, opts.MaxInputDimension,
		)
	}
	if cfg.Width > opts.MaxInputPixels/cfg.Height {
		return fmt.Errorf(
			"%w: dimensions %dx%d exceed %d pixels",
			ErrImageTooLarge, cfg.Width, cfg.Height, opts.MaxInputPixels,
		)
	}
	return nil
}

func transformImage(img image.Image, opts ImageOptions) image.Image {
	srcBounds := img.Bounds()
	if opts.CropSquare {
		srcBounds = centerSquare(srcBounds)
	}

	dstWidth, dstHeight := fitWithin(srcBounds.Dx(), srcBounds.Dy(), opts.MaxWidth, opts.MaxHeight)
	if srcBounds.Eq(img.Bounds()) && dstWidth == srcBounds.Dx() && dstHeight == srcBounds.Dy() {
		return img
	}

	dst := image.NewNRGBA(image.Rect(0, 0, dstWidth, dstHeight))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, srcBounds, xdraw.Over, nil)
	return dst
}

func centerSquare(rect image.Rectangle) image.Rectangle {
	width := rect.Dx()
	height := rect.Dy()
	if width == height {
		return rect
	}
	if width > height {
		offset := (width - height) / 2
		rect.Min.X += offset
		rect.Max.X = rect.Min.X + height
		return rect
	}
	offset := (height - width) / 2
	rect.Min.Y += offset
	rect.Max.Y = rect.Min.Y + width
	return rect
}

func fitWithin(width, height, maxWidth, maxHeight int) (int, int) {
	if maxWidth <= 0 {
		maxWidth = width
	}
	if maxHeight <= 0 {
		maxHeight = height
	}
	if width <= maxWidth && height <= maxHeight {
		return width, height
	}

	widthRatio := float64(maxWidth) / float64(width)
	heightRatio := float64(maxHeight) / float64(height)
	scale := min(widthRatio, heightRatio)
	return max(1, int(float64(width)*scale+0.5)),
		max(1, int(float64(height)*scale+0.5))
}
