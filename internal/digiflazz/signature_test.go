package digiflazz

import (
	"crypto/md5" //nolint:gosec // G501: MD5 required by Digiflazz API specification
	"encoding/hex"
	"strings"
	"testing"
)

func expectedMD5(input string) string {
	sum := md5.Sum([]byte(input)) //nolint:gosec // G401: MD5 required by Digiflazz API specification
	return hex.EncodeToString(sum[:])
}

func TestSignDepo(t *testing.T) {
	username := "testuser"
	apiKey := "testapikey"
	expected := expectedMD5(username + apiKey + "depo")
	if got := signDepo(username, apiKey); got != expected {
		t.Fatalf("signDepo() = %q, want %q", got, expected)
	}
}

func TestSignPricelist(t *testing.T) {
	username := "testuser"
	apiKey := "testapikey"
	expected := expectedMD5(username + apiKey + "pricelist")
	if got := signPricelist(username, apiKey); got != expected {
		t.Fatalf("signPricelist() = %q, want %q", got, expected)
	}
}

func TestSignDeposit(t *testing.T) {
	username := "testuser"
	apiKey := "testapikey"
	expected := expectedMD5(username + apiKey + "deposit")
	if got := signDeposit(username, apiKey); got != expected {
		t.Fatalf("signDeposit() = %q, want %q", got, expected)
	}
}

func TestSignTransaction(t *testing.T) {
	username := "testuser"
	apiKey := "testapikey"
	refID := "INV-20260513-001"
	expected := expectedMD5(username + apiKey + refID)
	if got := signTransaction(username, apiKey, refID); got != expected {
		t.Fatalf("signTransaction() = %q, want %q", got, expected)
	}
}

func TestSignInquiryPLN(t *testing.T) {
	username := "testuser"
	apiKey := "testapikey"
	customerNo := "1234554321"
	expected := expectedMD5(username + apiKey + customerNo)
	if got := signInquiryPLN(username, apiKey, customerNo); got != expected {
		t.Fatalf("signInquiryPLN() = %q, want %q", got, expected)
	}
}

func TestSignOutputFormat(t *testing.T) {
	got := signDepo("testuser", "testapikey")
	if len(got) != 32 {
		t.Fatalf("len(signDepo()) = %d, want 32", len(got))
	}
	if got != strings.ToLower(got) {
		t.Fatalf("signDepo() = %q, want lowercase hex", got)
	}
}
