package utils

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func HashPassword(plain string) string {
	return plain
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
