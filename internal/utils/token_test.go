package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestGenerateWebhookToken(t *testing.T) {
	token := GenerateWebhookToken()
	if len(token) != 64 {
		t.Fatalf("GenerateWebhookToken() length = %d, want 64", len(token))
	}
	if _, err := hex.DecodeString(token); err != nil {
		t.Fatalf("GenerateWebhookToken() not hex: %v", err)
	}
}

func TestHashToken(t *testing.T) {
	token := "webhook-token"
	want := sha256.Sum256([]byte(token))
	if got := HashToken(token); got != hex.EncodeToString(want[:]) {
		t.Fatalf("HashToken() = %q, want %q", got, hex.EncodeToString(want[:]))
	}
}

func TestVerifyToken(t *testing.T) {
	token := "webhook-token"
	hash := HashToken(token)
	if !VerifyToken(token, hash) {
		t.Fatal("VerifyToken() = false, want true")
	}
	if VerifyToken(token+"x", hash) {
		t.Fatal("VerifyToken() = true for wrong token")
	}
}
