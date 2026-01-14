package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

const (
	saltLength = 16
	keyLength  = 32
	iterations = 3
	memory     = 64 << 10 // ~64MB
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

	claimsAccess := jwt.RegisteredClaims{
		Issuer:    strconv.Itoa(userID),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}

	claimsRefresh := jwt.RegisteredClaims{
		Issuer:    strconv.Itoa(userID),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
	}

	jwtAccess := jwt.NewWithClaims(jwt.SigningMethodHS512, claimsAccess)
	jwtRefresh := jwt.NewWithClaims(jwt.SigningMethodHS512, claimsRefresh)

	access, err = jwtAccess.SignedString(secret)
	if err != nil {
		return "", "", nil
	}

	refresh, err = jwtRefresh.SignedString(secret)
	if err != nil {
		return "", "", nil
	}

	return access, refresh, nil
}

func SaveAsFile(f io.Reader, fname string) (err error) {
	content, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	fd, err := os.Create(fname)
	if err != nil {
		return fmt.Errorf("can't create file: %w", err)
	}
	defer func() {
		err := fd.Close()
		if err != nil {
			slog.Error("can't close resource", "err", err)
		}
	}()
	_, err = fd.Write(content)
	if err != nil {
		return fmt.Errorf("can't write to file: %w", err)
	}
	return nil
}
