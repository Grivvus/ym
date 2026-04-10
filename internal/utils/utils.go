package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

const (
	saltLength      = 16
	keyLength       = 32
	iterations      = 3
	memory          = 64 << 10 // ~64MB
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

var threads = uint8(runtime.NumCPU())

func generateSalt(length int) []byte {
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		panic("never")
	}

	return salt
}

func HashPassword(password string) (hashed []byte, salt []byte) {
	salt = generateSalt(saltLength)

	hashed = argon2.IDKey([]byte(password), salt, iterations, memory, threads, keyLength)

	return hashed, salt
}

func VerifyPassword(password string, salt []byte, expectedHash []byte) bool {
	newHash := argon2.IDKey([]byte(password), salt, iterations, memory, threads, keyLength)
	return subtle.ConstantTimeCompare(newHash, expectedHash) == 1
}

func CreateTokens(
	userID int, secret []byte,
) (access string, refresh string, err error) {
	now := time.Now()
	userIDStr := strconv.Itoa(userID)

	claimsAccess := jwt.RegisteredClaims{
		Subject:   userIDStr,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
	}

	claimsRefresh := jwt.RegisteredClaims{
		Subject:   userIDStr,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenTTL)),
	}

	jwtAccess := jwt.NewWithClaims(jwt.SigningMethodHS512, claimsAccess)
	jwtRefresh := jwt.NewWithClaims(jwt.SigningMethodHS512, claimsRefresh)

	access, err = jwtAccess.SignedString(deriveTokenSecret(secret, "access"))
	if err != nil {
		return "", "", fmt.Errorf("can't sign access token: %w", err)
	}

	refresh, err = jwtRefresh.SignedString(deriveTokenSecret(secret, "refresh"))
	if err != nil {
		return "", "", fmt.Errorf("can't sign refresh token: %w", err)
	}

	return access, refresh, nil
}

func ParseAccessToken(raw string, secret []byte) (int32, error) {
	return parseTokenUserID(raw, secret, "access")
}

func ParseRefreshToken(raw string, secret []byte) (int32, error) {
	return parseTokenUserID(raw, secret, "refresh")
}

func parseTokenUserID(raw string, secret []byte, tokenKind string) (int32, error) {
	claims := jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(
		raw, &claims,
		func(token *jwt.Token) (any, error) {
			alg := "<nil>"
			if token.Method != nil {
				alg = token.Method.Alg()
			}
			if token.Method == nil || alg != jwt.SigningMethodHS512.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", alg)
			}
			return deriveTokenSecret(secret, tokenKind), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}),
	)
	if err != nil {
		return 0, fmt.Errorf("can't parse %s token: %w", tokenKind, err)
	}

	userIDValue := claims.Subject
	if userIDValue == "" {
		userIDValue = claims.Issuer
	}
	if userIDValue == "" {
		return 0, errors.New("token does not contain user id")
	}

	parsedUserID, err := strconv.ParseInt(userIDValue, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid user id in token: %w", err)
	}

	return int32(parsedUserID), nil
}

func deriveTokenSecret(secret []byte, tokenKind string) []byte {
	derived := make([]byte, 0, len(secret)+1+len(tokenKind))
	derived = append(derived, secret...)
	derived = append(derived, ':')
	derived = append(derived, tokenKind...)
	return derived
}
