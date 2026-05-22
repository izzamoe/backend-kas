package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
)

// GenerateWebhookToken returns a secure random 32-byte hex token.
func GenerateWebhookToken() string {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

// HashString returns the hex-encoded SHA-256 hash of a string.
func HashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// HashToken returns the hex-encoded SHA-256 hash of a token.
func HashToken(token string) string {
	return HashString(token)
}

// VerifyToken compares a token against a stored SHA-256 hash.
func VerifyToken(token, hash string) bool {
	computed := HashToken(token)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}
