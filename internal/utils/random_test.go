package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/utils"
)

func TestGenerateRandomToken(t *testing.T) {
	tok, err := utils.GenerateRandomToken(16)
	require.NoError(t, err)
	assert.Len(t, tok, 32) // 16 bytes hex-encoded

	tok2, err := utils.GenerateRandomToken(16)
	require.NoError(t, err)
	assert.NotEqual(t, tok, tok2, "two calls must not collide")
}

func TestGenerateRandomToken_ZeroLength(t *testing.T) {
	tok, err := utils.GenerateRandomToken(0)
	require.NoError(t, err)
	assert.Empty(t, tok)
}
