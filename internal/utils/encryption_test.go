package utils

import (
	"encoding/base64"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := []byte("super-secret-passphrase")
	plaintext := "hello world"

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if ciphertext == "" {
		t.Fatal("Encrypt() returned empty ciphertext")
	}

	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		t.Fatalf("ciphertext is not base64: %v", err)
	}
	if len(decoded) <= encryptionSaltSize+encryptionNonceSize {
		t.Fatalf("ciphertext payload too short: %d", len(decoded))
	}

	got, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if got != plaintext {
		t.Fatalf("Decrypt() = %q, want %q", got, plaintext)
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	ciphertext, err := Encrypt("secret", []byte("key-one"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if _, err := Decrypt(ciphertext, []byte("key-two")); err == nil {
		t.Fatal("Decrypt() with wrong key expected error")
	}
}

func TestDecryptInvalidInputFails(t *testing.T) {
	if _, err := Decrypt("not-base64", []byte("key")); err == nil {
		t.Fatal("Decrypt() expected error for invalid base64")
	}
}
