package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT payload issued at login.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a signed HS256 JWT for the given user, expiring after expiry.
func GenerateJWT(userID, email, secret string, expiry time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyJWT parses and validates a JWT string, returning its claims.
func VerifyJWT(tokenString, secret string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// AdminClaims represents the JWT payload issued at admin login.
type AdminClaims struct {
	AdminID string `json:"admin_id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAdminJWT creates a signed HS256 JWT for the given admin, expiring after expiry.
func GenerateAdminJWT(adminID, email, role, secret string, expiry time.Duration) (string, error) {
	claims := AdminClaims{
		AdminID: adminID,
		Email:   email,
		Role:    role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyAdminJWT parses and validates an admin JWT string, returning its claims.
func VerifyAdminJWT(tokenString, secret string) (*AdminClaims, error) {
	claims := &AdminClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// refreshTokenBytes is the amount of entropy (in bytes) for a raw refresh
// token - 32 bytes (256 bits) encoded as hex, well above what's brute-forceable.
const refreshTokenBytes = 32

// GenerateRefreshToken returns a new random, URL-safe raw refresh token.
// This is an opaque credential, not a JWT - it carries no claims itself,
// it's just a lookup key for the refresh_tokens table.
func GenerateRefreshToken() (string, error) {
	return GenerateRandomToken(refreshTokenBytes)
}

// HashToken returns the SHA-256 hex digest of a raw token, for storing
// refresh tokens at rest without keeping the usable secret in the database.
func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
