package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/utils"
)

func TestHashPasswordAndCheckPassword(t *testing.T) {
	hash, err := utils.HashPassword("correct-horse-battery-staple")
	require.NoError(t, err)
	assert.NotEqual(t, "correct-horse-battery-staple", hash)

	assert.True(t, utils.CheckPassword("correct-horse-battery-staple", hash))
	assert.False(t, utils.CheckPassword("wrong-password", hash))
}

func TestHashPassword_DifferentHashesForSameInput(t *testing.T) {
	// bcrypt salts each hash, so two hashes of the same password must differ
	// while both still verifying successfully.
	hash1, err := utils.HashPassword("same-password")
	require.NoError(t, err)
	hash2, err := utils.HashPassword("same-password")
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2)
	assert.True(t, utils.CheckPassword("same-password", hash1))
	assert.True(t, utils.CheckPassword("same-password", hash2))
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	assert.False(t, utils.CheckPassword("anything", "not-a-bcrypt-hash"))
}
