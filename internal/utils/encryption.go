package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

const (
	encryptionSaltSize   = 32
	encryptionNonceSize  = 12
	encryptionKeySize    = 32
	encryptionIterations = 100000
)

// Encrypt encrypts plaintext with a key-derived AES-256-GCM key.
// Output format: base64(salt || nonce || ciphertext).
func Encrypt(plaintext string, key []byte) (string, error) {
	salt := make([]byte, encryptionSaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	derivedKey := pbkdf2SHA256(key, salt, encryptionIterations, encryptionKeySize)
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, encryptionNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aead.Seal(nil, nonce, []byte(plaintext), nil)
	payload := make([]byte, 0, len(salt)+len(nonce)+len(ciphertext))
	payload = append(payload, salt...)
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)

	return base64.StdEncoding.EncodeToString(payload), nil
}

// Decrypt decrypts a base64 payload formatted as salt || nonce || ciphertext.
func Decrypt(ciphertext string, key []byte) (string, error) {
	payload, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	minLen := encryptionSaltSize + encryptionNonceSize
	if len(payload) < minLen {
		return "", errors.New("ciphertext too short")
	}

	salt := payload[:encryptionSaltSize]
	nonce := payload[encryptionSaltSize:minLen]
	enc := payload[minLen:]

	derivedKey := pbkdf2SHA256(key, salt, encryptionIterations, encryptionKeySize)
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := aead.Open(nil, nonce, enc, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	if iterations <= 0 || keyLen <= 0 {
		return nil
	}
	var (
		hLen    = sha256.Size
		nBlocks = (keyLen + hLen - 1) / hLen
		dk      = make([]byte, 0, nBlocks*hLen)
	)

	for block := 1; block <= nBlocks; block++ {
		t := pbkdf2BlockSHA256(password, salt, iterations, block)
		dk = append(dk, t...)
	}

	return dk[:keyLen]
}

func pbkdf2BlockSHA256(password, salt []byte, iterations, blockNum int) []byte {
	blockIndex := []byte{byte(blockNum >> 24), byte(blockNum >> 16), byte(blockNum >> 8), byte(blockNum)}
	mac := hmac.New(sha256.New, password)
	mac.Write(salt)
	mac.Write(blockIndex)
	u := mac.Sum(nil)
	t := make([]byte, len(u))
	copy(t, u)

	for i := 1; i < iterations; i++ {
		mac = hmac.New(sha256.New, password)
		mac.Write(u)
		u = mac.Sum(nil)
		for j := range t {
			t[j] ^= u[j]
		}
	}

	return t
}
