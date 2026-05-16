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
	SaltLength                = 16
	DefaultPasswordKeyLength  = 32
	DefaultPasswordIterations = 3
	DefaultPasswordMemory     = 64 << 10 // ~64MB
	maxPasswordParallelism    = 4
	maxLegacyParallelism      = 32
	accessTokenTTL            = 15 * time.Minute
	refreshTokenTTL           = 30 * 24 * time.Hour
)

type PasswordHashParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	KeyLength   uint32
}

type tokenClaims struct {
	RefreshVersion int32 `json:"refresh_version,omitempty"`
	jwt.RegisteredClaims
}

func DefaultPasswordHashParams() PasswordHashParams {
	parallelism := runtime.NumCPU()
	if parallelism < 1 {
		parallelism = 1
	}
	if parallelism > maxPasswordParallelism {
		parallelism = maxPasswordParallelism
	}

	return PasswordHashParams{
		Memory:      DefaultPasswordMemory,
		Iterations:  DefaultPasswordIterations,
		Parallelism: uint8(parallelism),
		KeyLength:   DefaultPasswordKeyLength,
	}
}

func LegacyPasswordHashParams() PasswordHashParams {
	params := DefaultPasswordHashParams()
	params.Parallelism = 0
	return params
}

func generateSalt(length int) []byte {
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		panic("never")
	}

	return salt
}

func HashPassword(password string) (hashed []byte, salt []byte, params PasswordHashParams) {
	params = DefaultPasswordHashParams()
	salt = generateSalt(SaltLength)
	hashed = passwordHash(password, salt, params)

	return hashed, salt, params
}

func VerifyPassword(
	password string, salt []byte, expectedHash []byte, params PasswordHashParams,
) (bool, PasswordHashParams) {
	params = normalizePasswordHashParams(params)
	if params.Parallelism != 0 {
		return verifyPasswordWithParams(password, salt, expectedHash, params), params
	}

	for _, candidateParallelism := range legacyPasswordParallelismCandidates() {
		candidate := params
		candidate.Parallelism = candidateParallelism
		if verifyPasswordWithParams(password, salt, expectedHash, candidate) {
			return true, candidate
		}
	}
	return false, params
}

func normalizePasswordHashParams(params PasswordHashParams) PasswordHashParams {
	if params.Memory == 0 {
		params.Memory = DefaultPasswordMemory
	}
	if params.Iterations == 0 {
		params.Iterations = DefaultPasswordIterations
	}
	if params.KeyLength == 0 {
		params.KeyLength = DefaultPasswordKeyLength
	}
	return params
}

func verifyPasswordWithParams(
	password string, salt []byte, expectedHash []byte, params PasswordHashParams,
) bool {
	newHash := passwordHash(password, salt, params)
	return subtle.ConstantTimeCompare(newHash, expectedHash) == 1
}

func passwordHash(password string, salt []byte, params PasswordHashParams) []byte {
	return argon2.IDKey(
		[]byte(password), salt, params.Iterations,
		params.Memory, params.Parallelism, params.KeyLength,
	)
}

func legacyPasswordParallelismCandidates() []uint8 {
	seen := make(map[uint8]struct{}, maxLegacyParallelism)
	candidates := make([]uint8, 0, maxLegacyParallelism)
	add := func(value int) {
		if value <= 0 || value > 255 {
			return
		}
		candidate := uint8(value)
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}

	add(runtime.NumCPU())
	add(int(DefaultPasswordHashParams().Parallelism))
	for parallelism := 1; parallelism <= maxLegacyParallelism; parallelism++ {
		add(parallelism)
	}
	return candidates
}

func CreateTokens(
	userID int, secret []byte,
) (access string, refresh string, err error) {
	return CreateTokensWithRefreshVersion(userID, 0, secret)
}

func CreateTokensWithRefreshVersion(
	userID int, refreshVersion int32, secret []byte,
) (access string, refresh string, err error) {
	now := time.Now()
	userIDStr := strconv.Itoa(userID)

	claimsAccess := tokenClaims{
		RefreshVersion: refreshVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userIDStr,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
		},
	}

	claimsRefresh := tokenClaims{
		RefreshVersion: refreshVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userIDStr,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenTTL)),
		},
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
	userID, _, err := ParseRefreshTokenWithVersion(raw, secret)
	return userID, err
}

func ParseRefreshTokenWithVersion(raw string, secret []byte) (int32, int32, error) {
	userID, claims, err := parseTokenClaims(raw, secret, "refresh")
	if err != nil {
		return 0, 0, err
	}

	return userID, claims.RefreshVersion, nil
}

func parseTokenUserID(raw string, secret []byte, tokenKind string) (int32, error) {
	userID, _, err := parseTokenClaims(raw, secret, tokenKind)
	return userID, err
}

func parseTokenClaims(raw string, secret []byte, tokenKind string) (int32, tokenClaims, error) {
	claims := tokenClaims{}
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
		return 0, tokenClaims{}, fmt.Errorf("can't parse %s token: %w", tokenKind, err)
	}

	userIDValue := claims.Subject
	if userIDValue == "" {
		userIDValue = claims.Issuer
	}
	if userIDValue == "" {
		return 0, tokenClaims{}, errors.New("token does not contain user id")
	}

	parsedUserID, err := strconv.ParseInt(userIDValue, 10, 32)
	if err != nil {
		return 0, tokenClaims{}, fmt.Errorf("invalid user id in token: %w", err)
	}

	return int32(parsedUserID), claims, nil
}

func deriveTokenSecret(secret []byte, tokenKind string) []byte {
	derived := make([]byte, 0, len(secret)+1+len(tokenKind))
	derived = append(derived, secret...)
	derived = append(derived, ':')
	derived = append(derived, tokenKind...)
	return derived
}
