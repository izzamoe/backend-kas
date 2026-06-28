package digiflazz

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

const (
	EventCreate = "create"
	EventUpdate = "update"
)

type WebhookPayload struct {
	RefID          string            `json:"ref_id"`
	CustomerNo     string            `json:"customer_no"`
	CustomerName   string            `json:"customer_name,omitempty"`
	BuyerSKUCode   string            `json:"buyer_sku_code"`
	Message        string            `json:"message"`
	Status         TransactionStatus `json:"status"`
	Rc             string            `json:"rc"`
	Sn             string            `json:"sn,omitempty"`
	BuyerLastSaldo float64           `json:"buyer_last_saldo,omitempty"`
	Price          float64           `json:"price"`
	SellingPrice   float64           `json:"selling_price,omitempty"`
	Tele           string            `json:"tele,omitempty"`
	Wa             string            `json:"wa,omitempty"`
	Desc           json.RawMessage   `json:"desc,omitempty"`
}

type webhookPayloadAlias WebhookPayload

func (p *WebhookPayload) UnmarshalJSON(data []byte) error {
	type envelope struct {
		Data *webhookPayloadAlias `json:"data"`
		webhookPayloadAlias
	}

	var outer envelope
	if err := json.Unmarshal(data, &outer); err != nil {
		return err
	}

	if outer.Data != nil {
		*p = WebhookPayload(*outer.Data)
		if p.RefID != "" {
			return nil
		}
	}

	*p = WebhookPayload(outer.webhookPayloadAlias)
	return nil
}

type WebhookHeaders struct {
	Event     string
	Signature string
}

var (
	ErrInvalidSignatureFormat = errors.New("invalid signature format")
	ErrSignatureMismatch      = errors.New("signature mismatch")
)

func VerifyWebhookSignature(secret, signatureHeader string, rawBody []byte) error {
	if secret == "" {
		return nil
	}

	if !strings.HasPrefix(signatureHeader, "sha1=") {
		return ErrInvalidSignatureFormat
	}

	expectedHex := strings.TrimPrefix(signatureHeader, "sha1=")
	expected, err := hex.DecodeString(expectedHex)
	if err != nil {
		return ErrInvalidSignatureFormat
	}

	mac := hmac.New(sha1.New, []byte(secret))
	if _, err := mac.Write(rawBody); err != nil {
		return err
	}
	computed := mac.Sum(nil)

	if !hmac.Equal(computed, expected) {
		return ErrSignatureMismatch
	}

	return nil
}
