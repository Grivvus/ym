package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

func SHA256HexFromReadSeeker(r io.ReadSeeker) (string, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek to beginning before checksum: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, r); err != nil {
		return "", fmt.Errorf("calculate sha256 checksum: %w", err)
	}

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek to beginning after checksum: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
