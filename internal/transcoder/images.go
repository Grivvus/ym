package transcoder

import (
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

func FromBase64(r io.Reader) (io.Reader, error) {
	decoded := base64.NewDecoder(base64.StdEncoding, r)
	img, _, err := image.Decode(decoded)
	if err != nil {
		return nil, fmt.Errorf("can't transcode given image to webp: %w", err)
	}
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		if err := webp.Encode(pw, img, &encoder.Options{Lossless: true, ThreadLevel: true}); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()

	return pr, nil
}
