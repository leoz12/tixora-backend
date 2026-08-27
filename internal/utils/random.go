package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateRandomToken returns a random, URL-safe hex string with the given
// amount of entropy (in bytes) - used for refresh tokens and CSRF tokens.
func GenerateRandomToken(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
