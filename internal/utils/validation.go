package utils

import (
	"fmt"
	"net/mail"
	"strings"
)

func NormalizeEmailAddress(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("email is required")
	}

	parsed, err := mail.ParseAddress(trimmed)
	if err != nil || parsed.Address != trimmed {
		return "", fmt.Errorf("invalid email")
	}

	return strings.ToLower(trimmed), nil
}
