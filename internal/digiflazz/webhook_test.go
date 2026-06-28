package digiflazz

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // G505: SHA1 required by Digiflazz webhook signature
	"encoding/hex"
	"errors"
	"testing"
)

func testSignature(secret string, rawBody []byte) string {
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write(rawBody)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	rawBody := []byte(`{"data":{"ref_id":"test","status":"Sukses"}}`)
	expected := testSignature("mysecret", rawBody)

	if err := VerifyWebhookSignature("mysecret", "sha1="+expected, rawBody); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestVerifyWebhookSignature_Invalid(t *testing.T) {
	rawBody := []byte(`{"data":{"ref_id":"test","status":"Sukses"}}`)
	signature := testSignature("mysecret", rawBody)

	err := VerifyWebhookSignature("wrongsecret", "sha1="+signature, rawBody)
	if !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("expected ErrSignatureMismatch, got %v", err)
	}
}

func TestVerifyWebhookSignature_MissingPrefix(t *testing.T) {
	rawBody := []byte(`{"data":{"ref_id":"test","status":"Sukses"}}`)
	signature := testSignature("mysecret", rawBody)

	err := VerifyWebhookSignature("mysecret", signature, rawBody)
	if !errors.Is(err, ErrInvalidSignatureFormat) {
		t.Fatalf("expected ErrInvalidSignatureFormat, got %v", err)
	}
}

func TestVerifyWebhookSignature_EmptySecret(t *testing.T) {
	rawBody := []byte(`{"data":{"ref_id":"test","status":"Sukses"}}`)

	if err := VerifyWebhookSignature("", "sha1=whatever", rawBody); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
