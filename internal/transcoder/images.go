package transcoder

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

func ToWebp(r io.Reader) (io.ReadCloser, error) {
	img, formatFound, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("can't decode image: %w", err)
	}
	slog.Info("transcoder", "image format detected", formatFound)
	pr, pw := io.Pipe()

	opts, err := encoder.NewLosslessEncoderOptions(encoder.PresetDefault, 0)
	if err != nil {
		return nil, fmt.Errorf("can't init webp encoder options: %w", err)
	}
	go func() {
		defer pw.Close()
		// defer func() {
		// 	if r := recover(); r != nil {
		// 		slog.Error("image transcoder goroutine was recovered", "err", r)
		// 	}
		// }()

		if err := webp.Encode(pw, img, opts); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()

	return pr, nil
}
