package transcoder

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

var ErrCantDecode = errors.New("can't decode media")

var ErrCantEncode = errors.New("can't encode media")

func ToWebp(r io.Reader) (io.ReadCloser, error) {
	img, formatFound, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCantDecode, err)
	}
	slog.Info("transcoder", "image format detected", formatFound)
	pr, pw := io.Pipe()

	opts, err := encoder.NewLosslessEncoderOptions(encoder.PresetDefault, 0)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCantEncode, err)
	}
	go func() {
		defer func() { _ = pw.Close() }()
		if err := webp.Encode(pw, img, opts); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("%w: %w", ErrCantEncode, err))
			return
		}
	}()

	return pr, nil
}
