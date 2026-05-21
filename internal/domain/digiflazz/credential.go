package digiflazz

import "time"

type CredentialDTO struct {
	ID                string    `json:"id"`
	FamilyID          string    `json:"family_id"`
	Username          string    `json:"username"`
	APIKeyLast4       string    `json:"api_key_last4,omitempty"`
	Testing           bool      `json:"testing"`
	IsActive          bool      `json:"is_active"`
	Balance           *float64  `json:"balance,omitempty"`
	WebhookConfigured bool      `json:"webhook_configured"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CreateCredentialRequest struct {
	FamilyID      string `json:"family_id"`
	Username      string `json:"username"`
	APIKey        string `json:"api_key"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
	Testing       bool   `json:"testing,omitempty"`
}

type UpdateCredentialRequest struct {
	Username      *string `json:"username,omitempty"`
	APIKey        *string `json:"api_key,omitempty"`
	WebhookSecret *string `json:"webhook_secret,omitempty"`
	Testing       *bool   `json:"testing,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
}

type BalanceResponse struct {
	FamilyID string  `json:"family_id"`
	Balance  float64 `json:"balance"`
}

type RotateWebhookTokenResponse struct {
	Credential *CredentialDTO `json:"credential"`
	Token      string         `json:"token"`
}

type DigiflazzDepositResponse struct {
	Rc            string  `json:"rc"`
	Bank          string  `json:"bank"`
	PaymentMethod string  `json:"payment_method"`
	AccountNo     string  `json:"account_no"`
	Notes         string  `json:"notes"`
	Amount        float64 `json:"amount"`
}

// UpsertCredentialRequest is the unified request body for POST /credential (create or update).
// FamilyID comes from middleware — do NOT put it here.
type UpsertCredentialRequest struct {
	Username      string  `json:"username"`                 // required
	APIKey        string  `json:"api_key"`                  // required
	WebhookSecret *string `json:"webhook_secret,omitempty"` // nil = keep existing; &"" = clear
	Testing       *bool   `json:"testing,omitempty"`        // nil = keep existing
}

// UpsertCredentialResult is returned by the credential service UPSERT operation.
type UpsertCredentialResult struct {
	Credential      *CredentialDTO
	RawWebhookToken string // non-empty only on CREATE path; empty on UPDATE
	SyncInitiated   bool   // always true after successful UPSERT
}
