package repository

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"kas/generated"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const digiflazzCredentialsCollection = "digiflazz_credentials"

type DigiflazzCredentialRepository interface {
	Create(data *DigiflazzCredentialCreateData) (*digiflazzdomain.CredentialDTO, error)
	GetByID(id string) (*digiflazzdomain.CredentialDTO, error)
	GetSecretByID(id string) (*DigiflazzCredentialRecord, error)
	GetByFamilyID(familyID string) (*digiflazzdomain.CredentialDTO, error)
	GetSecretByFamilyID(familyID string) (*DigiflazzCredentialRecord, error)
	GetActiveByFamilyID(familyID string) (*digiflazzdomain.CredentialDTO, error)
	GetActiveSecretByFamilyID(familyID string) (*DigiflazzCredentialRecord, error)
	GetSecretByWebhookTokenHash(tokenHash string) (*DigiflazzCredentialRecord, error)
	ListByFamilyID(familyID string, limit, offset int) ([]*digiflazzdomain.CredentialDTO, error)
	CountByFamilyID(familyID string) (int, error)
	Update(id string, data *DigiflazzCredentialUpdateData) (*digiflazzdomain.CredentialDTO, error)
	Disable(id string) (*digiflazzdomain.CredentialDTO, error)
	Delete(id string) error
	ListAllActive() ([]*DigiflazzCredentialRecord, error)
}

type digiflazzCredentialRepo struct {
	app core.App
}

func NewDigiflazzCredentialRepository(app core.App) DigiflazzCredentialRepository {
	return &digiflazzCredentialRepo{app: app}
}

type DigiflazzCredentialCreateData struct {
	FamilyID         string
	Username         string
	APIKeyCiphertext string
	APIKeyLast4      string
	APIKeyHash       string
	WebhookID        string
	WebhookTokenHash string
	WebhookSecret    string
	Testing          bool
	IsActive         bool
}

type DigiflazzCredentialUpdateData struct {
	Username         *string
	APIKeyCiphertext *string
	APIKeyLast4      *string
	APIKeyHash       *string
	WebhookID        *string
	WebhookTokenHash *string
	WebhookSecret    *string
	Testing          *bool
	IsActive         *bool
}

type DigiflazzCredentialRecord struct {
	ID               string
	FamilyID         string
	Username         string
	APIKeyCiphertext string
	APIKeyLast4      string
	APIKeyHash       string
	WebhookID        string
	WebhookTokenHash string
	WebhookSecret    string
	Testing          bool
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type digiflazzCredentialSecretPayload struct {
	Ciphertext       string `json:"ciphertext"`
	Last4            string `json:"last4,omitempty"`
	Hash             string `json:"hash,omitempty"`
	WebhookTokenHash string `json:"webhook_token_hash,omitempty"`
}

func (r *digiflazzCredentialRepo) Create(data *DigiflazzCredentialCreateData) (*digiflazzdomain.CredentialDTO, error) {
	proxy, err := generated.NewProxy[generated.DigiflazzCredentials](r.app)
	if err != nil {
		return nil, err
	}

	proxy.Record.Set("family_id", data.FamilyID)
	proxy.SetUsername(data.Username)
	proxy.SetApiKey(encodeDigiflazzCredentialSecret(data.APIKeyCiphertext, data.APIKeyLast4, data.APIKeyHash, data.WebhookTokenHash))
	proxy.SetWebhookId(data.WebhookID)
	proxy.Record.Set("webhook_token_hash", data.WebhookTokenHash)
	proxy.Record.Set("webhook_secret", data.WebhookSecret)
	proxy.SetTesting(data.Testing)
	proxy.SetIsActive(data.IsActive)

	if err := r.app.Save(proxy.Record); err != nil {
		return nil, err
	}

	return r.recordToDTO(proxy.Record)
}

func (r *digiflazzCredentialRepo) GetByID(id string) (*digiflazzdomain.CredentialDTO, error) {
	record, err := r.app.FindRecordById(digiflazzCredentialsCollection, id)
	if err != nil {
		return nil, err
	}
	return r.recordToDTO(record)
}

func (r *digiflazzCredentialRepo) GetSecretByID(id string) (*DigiflazzCredentialRecord, error) {
	record, err := r.app.FindRecordById(digiflazzCredentialsCollection, id)
	if err != nil {
		return nil, err
	}
	return r.recordToRecord(record)
}

func (r *digiflazzCredentialRepo) GetByFamilyID(familyID string) (*digiflazzdomain.CredentialDTO, error) {
	record, err := r.findFirstByFamilyFilter("family_id = {:familyID}", familyID)
	if err != nil || record == nil {
		return nil, err
	}
	return r.recordToDTO(record)
}

func (r *digiflazzCredentialRepo) GetSecretByFamilyID(familyID string) (*DigiflazzCredentialRecord, error) {
	record, err := r.findFirstByFamilyFilter("family_id = {:familyID}", familyID)
	if err != nil || record == nil {
		return nil, err
	}
	return r.recordToRecord(record)
}

func (r *digiflazzCredentialRepo) GetActiveByFamilyID(familyID string) (*digiflazzdomain.CredentialDTO, error) {
	record, err := r.findFirstByFamilyFilter("family_id = {:familyID} && is_active = true", familyID)
	if err != nil || record == nil {
		return nil, err
	}
	return r.recordToDTO(record)
}

func (r *digiflazzCredentialRepo) GetActiveSecretByFamilyID(familyID string) (*DigiflazzCredentialRecord, error) {
	record, err := r.findFirstByFamilyFilter("family_id = {:familyID} && is_active = true", familyID)
	if err != nil || record == nil {
		return nil, err
	}
	return r.recordToRecord(record)
}

func (r *digiflazzCredentialRepo) GetSecretByWebhookTokenHash(tokenHash string) (*DigiflazzCredentialRecord, error) {
	if tokenHash == "" {
		return nil, nil
	}

	record, err := r.app.FindFirstRecordByFilter(
		digiflazzCredentialsCollection,
		"webhook_token_hash = {:tokenHash}",
		dbx.Params{"tokenHash": tokenHash},
	)
	if err == nil && record != nil {
		return r.recordToRecord(record)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	records, err := r.app.FindRecordsByFilter(digiflazzCredentialsCollection, "id != ''", "-created", -1, 0)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		credential, err := r.recordToRecord(record)
		if err != nil {
			return nil, err
		}
		if subtle.ConstantTimeCompare([]byte(credential.WebhookTokenHash), []byte(tokenHash)) == 1 {
			return credential, nil
		}
	}
	return nil, nil
}

func (r *digiflazzCredentialRepo) ListByFamilyID(familyID string, limit, offset int) ([]*digiflazzdomain.CredentialDTO, error) {
	records, err := r.app.FindRecordsByFilter(
		digiflazzCredentialsCollection,
		"family_id = {:familyID}",
		"-created",
		limit,
		offset,
		dbx.Params{"familyID": familyID},
	)
	if err != nil {
		return nil, err
	}

	dtos := make([]*digiflazzdomain.CredentialDTO, 0, len(records))
	for _, record := range records {
		dto, err := r.recordToDTO(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert credential %s: %w", record.Id, err)
		}
		dtos = append(dtos, dto)
	}
	return dtos, nil
}

func (r *digiflazzCredentialRepo) CountByFamilyID(familyID string) (int, error) {
	var count int
	err := r.app.DB().NewQuery("SELECT COUNT(*) FROM digiflazz_credentials WHERE family_id = {:familyID}").Bind(dbx.Params{
		"familyID": familyID,
	}).Row(&count)
	return count, err
}

func (r *digiflazzCredentialRepo) Update(id string, data *DigiflazzCredentialUpdateData) (*digiflazzdomain.CredentialDTO, error) {
	record, err := r.app.FindRecordById(digiflazzCredentialsCollection, id)
	if err != nil {
		return nil, err
	}
	proxy, err := generated.WrapRecord[generated.DigiflazzCredentials](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap digiflazz credential record: %w", err)
	}

	current := decodeDigiflazzCredentialSecret(proxy.ApiKey())
	if data.Username != nil {
		proxy.SetUsername(*data.Username)
	}
	if data.APIKeyCiphertext != nil {
		current.Ciphertext = *data.APIKeyCiphertext
	}
	if data.APIKeyLast4 != nil {
		current.Last4 = *data.APIKeyLast4
	}
	if data.APIKeyHash != nil {
		current.Hash = *data.APIKeyHash
	}
	if data.WebhookTokenHash != nil {
		current.WebhookTokenHash = *data.WebhookTokenHash
		proxy.Record.Set("webhook_token_hash", *data.WebhookTokenHash)
	}
	if data.WebhookID != nil {
		proxy.SetWebhookId(*data.WebhookID)
	}
	if data.WebhookSecret != nil {
		proxy.Record.Set("webhook_secret", *data.WebhookSecret)
	}
	if data.APIKeyCiphertext != nil || data.APIKeyLast4 != nil || data.APIKeyHash != nil || data.WebhookTokenHash != nil {
		proxy.SetApiKey(mustEncodeDigiflazzCredentialSecret(current))
	}
	if data.Testing != nil {
		proxy.SetTesting(*data.Testing)
	}
	if data.IsActive != nil {
		proxy.SetIsActive(*data.IsActive)
	}

	if err := r.app.Save(record); err != nil {
		return nil, err
	}
	return r.recordToDTO(record)
}

func (r *digiflazzCredentialRepo) Disable(id string) (*digiflazzdomain.CredentialDTO, error) {
	active := false
	return r.Update(id, &DigiflazzCredentialUpdateData{IsActive: &active})
}

func (r *digiflazzCredentialRepo) Delete(id string) error {
	record, err := r.app.FindRecordById(digiflazzCredentialsCollection, id)
	if err != nil {
		return err
	}
	return r.app.Delete(record)
}

func (r *digiflazzCredentialRepo) ListAllActive() ([]*DigiflazzCredentialRecord, error) {
	records, err := r.app.FindRecordsByFilter(
		digiflazzCredentialsCollection,
		"is_active = true",
		"-created",
		-1,
		0,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list active digiflazz credentials: %w", err)
	}
	results := make([]*DigiflazzCredentialRecord, 0, len(records))
	for _, record := range records {
		cred, err := r.recordToRecord(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert digiflazz credential %s: %w", record.Id, err)
		}
		results = append(results, cred)
	}
	return results, nil
}

func (r *digiflazzCredentialRepo) findFirstByFamilyFilter(filter, familyID string) (*core.Record, error) {
	record, err := r.app.FindFirstRecordByFilter(
		digiflazzCredentialsCollection,
		filter,
		dbx.Params{"familyID": familyID},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return record, nil
}

func (r *digiflazzCredentialRepo) recordToDTO(record *core.Record) (*digiflazzdomain.CredentialDTO, error) {
	cred, err := r.recordToRecord(record)
	if err != nil {
		return nil, err
	}
	return cred.SafeDTO(), nil
}

func (r *digiflazzCredentialRepo) recordToRecord(record *core.Record) (*DigiflazzCredentialRecord, error) {
	proxy, err := generated.WrapRecord[generated.DigiflazzCredentials](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap digiflazz credential record: %w", err)
	}
	payload := decodeDigiflazzCredentialSecret(proxy.ApiKey())
	webhookTokenHash := record.GetString("webhook_token_hash")
	if webhookTokenHash == "" {
		webhookTokenHash = payload.WebhookTokenHash
	}

	return &DigiflazzCredentialRecord{
		ID:               proxy.Id,
		FamilyID:         record.GetString("family_id"),
		Username:         proxy.Username(),
		APIKeyCiphertext: payload.Ciphertext,
		APIKeyLast4:      payload.Last4,
		APIKeyHash:       payload.Hash,
		WebhookID:        proxy.WebhookId(),
		WebhookTokenHash: webhookTokenHash,
		WebhookSecret:    record.GetString("webhook_secret"),
		Testing:          proxy.Testing(),
		IsActive:         proxy.IsActive(),
		CreatedAt:        proxy.Created().Time(),
		UpdatedAt:        proxy.Updated().Time(),
	}, nil
}

func (r *DigiflazzCredentialRecord) SafeDTO() *digiflazzdomain.CredentialDTO {
	return &digiflazzdomain.CredentialDTO{
		ID:                r.ID,
		FamilyID:          r.FamilyID,
		Username:          r.Username,
		APIKeyLast4:       r.APIKeyLast4,
		Testing:           r.Testing,
		IsActive:          r.IsActive,
		WebhookID:         r.WebhookID,
		WebhookConfigured: r.WebhookTokenHash != "",
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

func encodeDigiflazzCredentialSecret(ciphertext, last4, hash, webhookTokenHash string) string {
	return mustEncodeDigiflazzCredentialSecret(digiflazzCredentialSecretPayload{
		Ciphertext:       ciphertext,
		Last4:            last4,
		Hash:             hash,
		WebhookTokenHash: webhookTokenHash,
	})
}

func mustEncodeDigiflazzCredentialSecret(payload digiflazzCredentialSecretPayload) string {
	b, err := json.Marshal(payload)
	if err != nil {
		return payload.Ciphertext
	}
	return string(b)
}

func decodeDigiflazzCredentialSecret(value string) digiflazzCredentialSecretPayload {
	var payload digiflazzCredentialSecretPayload
	if err := json.Unmarshal([]byte(value), &payload); err == nil && payload.Ciphertext != "" {
		return payload
	}
	return digiflazzCredentialSecretPayload{Ciphertext: value}
}
