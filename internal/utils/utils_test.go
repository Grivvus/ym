package utils_test

import (
	"testing"

	"github.com/Grivvus/ym/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_HashNotEQPlain(t *testing.T) {
	t.Parallel()
	const plain = "password"
	hashed, salt, _ := utils.HashPassword(plain)
	assert.NotEqual(t, len(hashed), len(plain))
	assert.NotEqual(t, []byte(plain), hashed)
	assert.NotEqual(t, salt, hashed)
	assert.NotEqual(t, []byte(plain), salt)
}

func TestVerifyPassword_EqualPasswords(t *testing.T) {
	t.Parallel()
	const password = "password"
	hash, salt, params := utils.HashPassword(password)
	require.NotNil(t, hash)
	require.NotNil(t, salt)
	require.True(t, len(hash) > 0)
	require.True(t, len(salt) > 0)
	verified, _ := utils.VerifyPassword(password, salt, hash, params)
	assert.True(t, verified)
}

func TestVerifyPassword_NotEqualPasswords(t *testing.T) {
	t.Parallel()
	const password = "password"
	const password2 = "password2"
	hash, salt, params := utils.HashPassword(password)
	require.NotNil(t, hash)
	require.NotNil(t, salt)
	require.True(t, len(hash) > 0)
	require.True(t, len(salt) > 0)
	verified, _ := utils.VerifyPassword(password2, salt, hash, params)
	assert.False(t, verified)
}

func TestCreateTokens(t *testing.T) {
	t.Parallel()
	const userID = 11
	secret := []byte("secret")
	access, refresh, err := utils.CreateTokens(userID, secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
	assert.NotEqual(t, access, refresh)
}

func TestCreateTokens_EmptySecret(t *testing.T) {
	t.Parallel()
	const userID = 11
	secret := []byte("")
	access, refresh, err := utils.CreateTokens(userID, secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
	assert.NotEqual(t, access, refresh)
}

func TestCreateTokens_NilSecret(t *testing.T) {
	t.Parallel()
	const userID = 11
	var secret []byte
	access, refresh, err := utils.CreateTokens(userID, secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
	assert.NotEqual(t, access, refresh)
}

func TestParseAccessToken_ParseWithValidSecret(t *testing.T) {
	t.Parallel()
	const userID = 13
	secret := []byte("secret")
	access, _, err := utils.CreateTokens(userID, secret)
	require.NoError(t, err)
	parsedID, err := utils.ParseAccessToken(access, secret)
	assert.NoError(t, err)
	assert.Equal(t, int32(userID), parsedID)
}

func TestParseAccessToken_ParseWithInvalidSecret(t *testing.T) {
	t.Parallel()
	const userID = 13
	secret := []byte("secret")
	wrongSecret := []byte("wrongSecret")
	access, _, err := utils.CreateTokens(userID, secret)
	require.NoError(t, err)
	parsedID, err := utils.ParseAccessToken(access, wrongSecret)
	assert.Error(t, err)
	assert.NotEqual(t, int32(userID), parsedID)
}

func TestParseRefreshToken_ParseWithValidSecret(t *testing.T) {
	t.Parallel()
	const userID = 13
	secret := []byte("secret")
	_, refresh, err := utils.CreateTokens(userID, secret)
	require.NoError(t, err)
	parsedID, err := utils.ParseRefreshToken(refresh, secret)
	assert.NoError(t, err)
	assert.Equal(t, int32(userID), parsedID)
}

func TestParseRefreshToken_ParseWithInvalidSecret(t *testing.T) {
	t.Parallel()
	const userID = 13
	secret := []byte("secret")
	wrongSecret := []byte("wrongSecret")
	_, refresh, err := utils.CreateTokens(userID, secret)
	require.NoError(t, err)
	parsedID, err := utils.ParseRefreshToken(refresh, wrongSecret)
	assert.Error(t, err)
	assert.NotEqual(t, int32(userID), parsedID)
}

func TestParseRefreshTokenWithVersion_ParseWithValidSecret(t *testing.T) {
	t.Parallel()
	const userID = 13
	const refreshVersion = 7
	secret := []byte("secret")
	_, refresh, err := utils.CreateTokensWithRefreshVersion(userID, refreshVersion, secret)
	require.NoError(t, err)

	parsedID, parsedVersion, err := utils.ParseRefreshTokenWithVersion(refresh, secret)
	assert.NoError(t, err)
	assert.Equal(t, int32(userID), parsedID)
	assert.Equal(t, int32(refreshVersion), parsedVersion)
}
