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
	hashed, salt := utils.HashPassword(plain)
	assert.NotEqual(t, len(hashed), len(plain))
	assert.NotEqual(t, []byte(plain), hashed)
	assert.NotEqual(t, salt, hashed)
	assert.NotEqual(t, []byte(plain), salt)
}

func TestVerifyPassword_EqualPasswords(t *testing.T) {
	t.Parallel()
	const password = "password"
	hash, salt := utils.HashPassword(password)
	require.NotNil(t, hash)
	require.NotNil(t, salt)
	require.True(t, len(hash) > 0)
	require.True(t, len(salt) > 0)
	assert.True(t, utils.VerifyPassword(password, salt, hash))
}

func TestVerifyPassword_NotEqualPasswords(t *testing.T) {
	t.Parallel()
	const password = "password"
	const password2 = "password2"
	hash, salt := utils.HashPassword(password)
	require.NotNil(t, hash)
	require.NotNil(t, salt)
	require.True(t, len(hash) > 0)
	require.True(t, len(salt) > 0)
	assert.False(t, utils.VerifyPassword(password2, salt, hash))
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

// must clarify behavior

//func TestCreateTokens_EmptySecret(t *testing.T) {
//	t.Parallel()
//	const userID = 11
//	secret := []byte("")
//	access, refresh, err := utils.CreateTokens(userID, secret)
//	assert.NoError(t, err)
//	assert.NotEmpty(t, access)
//	assert.NotEmpty(t, refresh)
//	assert.NotEqual(t, access, refresh)
//}
//
//func TestCreateTokens_NilSecret(t *testing.T) {
//	t.Parallel()
//	const userID = 11
//	var secret []byte
//	access, refresh, err := utils.CreateTokens(userID, secret)
//	assert.NoError(t, err)
//	assert.NotEmpty(t, access)
//	assert.NotEmpty(t, refresh)
//	assert.NotEqual(t, access, refresh)
//}

func TestParseAccessToken_ParseWithValidSecret(t *testing.T) {}

func TestParseAccessToken_ParseWithInvalidSecret(t *testing.T) {}

func TestParseRefreshToken_ParseWithValidSecret(t *testing.T) {}

func TestParseRefreshToken_ParseWithInvalidSecret(t *testing.T) {}
