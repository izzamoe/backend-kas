package repository

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"kas/generated"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/utils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const digiflazzEventsCollection = "digiflazz_events"

var digiflazzEventExpandFields = []string{"order_id"}

var digiflazzEventSensitiveFields = []string{
	"token",
	"password",
	"api_key",
	"secret",
	"signature",
	"pin",
	"otp",
	"authorization",
}

type digiflazzEventResponseEnvelope struct {
	PayloadHash  string `json:"payload_hash,omitempty"`
	RC           string `json:"rc,omitempty"`
	Message      string `json:"message,omitempty"`
	SN           string `json:"sn,omitempty"`
	CreatedBy    string `json:"created_by,omitempty"`
	ProcessedAt  string `json:"processed_at,omitempty"`
	StatusBefore string `json:"status_before,omitempty"`
	StatusAfter  string `json:"status_after,omitempty"`
	Source       string `json:"source,omitempty"`
}

// DigiflazzEventRepository defines append-only access for audit events.
type DigiflazzEventRepository interface {
	Create(data *DigiflazzEventCreateData) (*DigiflazzEventRecord, error)
	GetByID(id string) (*DigiflazzEventRecord, error)
	GetByFamilyAndID(familyID, id string) (*DigiflazzEventRecord, error)
	ListByFamilyID(familyID string, limit, offset int) ([]*DigiflazzEventRecord, error)
	ExistsByOrderAndPayloadHash(orderID, payloadHash string) (bool, error)
}

// digiflazzEventRepo is the concrete PocketBase implementation.
type digiflazzEventRepo struct {
	app core.App
}

// NewDigiflazzEventRepository creates a new repository backed by PocketBase.
func NewDigiflazzEventRepository(app core.App) DigiflazzEventRepository {
	return &digiflazzEventRepo{app: app}
}

// DigiflazzEventCreateData carries the data needed to persist an audit event.
type DigiflazzEventCreateData struct {
	OrderID      string
	EventType    digiflazzdomain.EventType
	StatusBefore string
	StatusAfter  string
	Source       string
	RC           string
	Message      string
	SN           string
	Payload      string
	PayloadHash  string
	ProcessedAt  *time.Time
	CreatedBy    string
}

// DigiflazzEventRecord represents a persisted audit event.
type DigiflazzEventRecord struct {
	ID              string
	FamilyID        string
	OrderID         string
	EventType       digiflazzdomain.EventType
	StatusBefore    string
	StatusAfter     string
	Source          string
	RC              string
	Message         string
	SN              string
	RedactedPayload string
	PayloadHash     string
	ProcessedAt     *time.Time
	CreatedBy       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Create stores a redacted audit event and returns the existing record for duplicate payload hashes.
func (r *digiflazzEventRepo) Create(data *DigiflazzEventCreateData) (*DigiflazzEventRecord, error) {
	payloadHash := data.PayloadHash
	if payloadHash == "" {
		payloadHash = hashPayload(data.Payload)
	}

	var created *DigiflazzEventRecord
	err := r.app.RunInTransaction(func(txApp core.App) error {
		existing, err := r.findByOrderAndHash(txApp, data.OrderID, payloadHash)
		if err != nil {
			return err
		}
		if existing != nil {
			created = existing
			return nil
		}

		redactedPayload, err := redactDigiflazzPayload(data.Payload)
		if err != nil {
			return fmt.Errorf("failed to redact digiflazz payload: %w", err)
		}
		processedAt := processedAtOrNow(data.ProcessedAt)

		proxy, err := generated.NewProxy[generated.DigiflazzEvents](txApp)
		if err != nil {
			return fmt.Errorf("failed to create digiflazz event proxy: %w", err)
		}

		proxy.Record.Set("order_id", data.OrderID)
		if err := setDigiflazzEventType(proxy.Record, data.EventType); err != nil {
			return err
		}
		proxy.SetStatusBefore(data.StatusBefore)
		proxy.SetStatusAfter(data.StatusAfter)
		proxy.SetSource(data.Source)
		proxy.Record.Set("rc", data.RC)
		proxy.Record.Set("message", data.Message)
		proxy.Record.Set("sn", data.SN)
		proxy.SetPayload(redactedPayload)
		proxy.Record.Set("redacted_payload", redactedPayload)
		proxy.Record.Set("payload_hash", payloadHash)
		proxy.Record.Set("processed_at", processedAt)
		proxy.Record.Set("created_by", data.CreatedBy)
		proxy.SetResponse(mustMarshalDigiflazzEventResponse(digiflazzEventResponseEnvelope{
			PayloadHash:  payloadHash,
			RC:           data.RC,
			Message:      data.Message,
			SN:           data.SN,
			CreatedBy:    data.CreatedBy,
			ProcessedAt:  processedAt.String(),
			StatusBefore: data.StatusBefore,
			StatusAfter:  data.StatusAfter,
			Source:       data.Source,
		}))

		if err := txApp.Save(proxy.Record); err != nil {
			return fmt.Errorf("failed to save digiflazz event: %w", err)
		}

		txApp.ExpandRecord(proxy.Record, digiflazzEventExpandFields, nil)
		created, err = r.recordToRecord(proxy.Record)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *digiflazzEventRepo) ExistsByOrderAndPayloadHash(orderID, payloadHash string) (bool, error) {
	existing, err := r.findByOrderAndHash(r.app, orderID, payloadHash)
	if err != nil {
		return false, err
	}
	return existing != nil, nil
}

// GetByID retrieves an event by its record ID.
func (r *digiflazzEventRepo) GetByID(id string) (*DigiflazzEventRecord, error) {
	record, err := r.app.FindRecordById(digiflazzEventsCollection, id)
	if err != nil {
		if errorsIsNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find digiflazz event: %w", err)
	}

	r.app.ExpandRecord(record, digiflazzEventExpandFields, nil)
	return r.recordToRecord(record)
}

// GetByFamilyAndID retrieves an event only when it belongs to the given family.
func (r *digiflazzEventRepo) GetByFamilyAndID(familyID, id string) (*DigiflazzEventRecord, error) {
	record, err := r.findOneByFamilyFilter("id = {:id} && order_id.family_id = {:familyID}", familyID, dbx.Params{"id": id})
	if err != nil || record == nil {
		return nil, err
	}

	r.app.ExpandRecord(record, digiflazzEventExpandFields, nil)
	return r.recordToRecord(record)
}

// ListByFamilyID returns events for a family in descending creation order.
func (r *digiflazzEventRepo) ListByFamilyID(familyID string, limit, offset int) ([]*DigiflazzEventRecord, error) {
	records, err := r.app.FindRecordsByFilter(
		digiflazzEventsCollection,
		"order_id.family_id = {:familyID}",
		"-created",
		limit,
		offset,
		dbx.Params{"familyID": familyID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list digiflazz events: %w", err)
	}

	r.app.ExpandRecords(records, digiflazzEventExpandFields, nil)

	items := make([]*DigiflazzEventRecord, 0, len(records))
	for _, record := range records {
		item, err := r.recordToRecord(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert digiflazz event %s: %w", record.Id, err)
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *digiflazzEventRepo) findByOrderAndHash(app core.App, orderID, payloadHash string) (*DigiflazzEventRecord, error) {
	records, err := app.FindRecordsByFilter(
		digiflazzEventsCollection,
		"order_id = {:orderID}",
		"-created",
		-1,
		0,
		dbx.Params{"orderID": orderID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check digiflazz event duplicate: %w", err)
	}

	for _, record := range records {
		app.ExpandRecord(record, digiflazzEventExpandFields, nil)
		item, err := r.recordToRecord(record)
		if err != nil {
			return nil, err
		}
		if item.PayloadHash == payloadHash {
			return item, nil
		}
	}

	return nil, nil
}

func (r *digiflazzEventRepo) findOneByFamilyFilter(filter string, familyID string, params dbx.Params) (*core.Record, error) {
	if params == nil {
		params = dbx.Params{}
	}
	params["familyID"] = familyID
	record, err := r.app.FindFirstRecordByFilter(digiflazzEventsCollection, filter, params)
	if err != nil {
		if errorsIsNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find digiflazz event: %w", err)
	}
	return record, nil
}

func (r *digiflazzEventRepo) recordToRecord(record *core.Record) (*DigiflazzEventRecord, error) {
	proxy, err := generated.WrapRecord[generated.DigiflazzEvents](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap digiflazz event record: %w", err)
	}

	processedAt := record.GetDateTime("processed_at").Time()
	var processedAtPtr *time.Time
	if !processedAt.IsZero() {
		processedAt = processedAt.UTC()
		processedAtPtr = &processedAt
	}

	responseEnvelope := decodeDigiflazzEventResponse(record.GetString("response"))
	if processedAtPtr == nil && responseEnvelope.ProcessedAt != "" {
		if parsed, err := types.ParseDateTime(responseEnvelope.ProcessedAt); err == nil {
			t := parsed.Time().UTC()
			processedAtPtr = &t
		} else if parsedTime, err := time.Parse(time.RFC3339Nano, responseEnvelope.ProcessedAt); err == nil {
			parsedTime = parsedTime.UTC()
			processedAtPtr = &parsedTime
		}
	}

	familyID := ""
	if order := proxy.OrderId(); order != nil {
		familyID = order.Record.GetString("family_id")
	}

	payloadHash := firstNonEmpty(record.GetString("payload_hash"), responseEnvelope.PayloadHash)
	if payloadHash == "" {
		payloadHash = hashPayload(proxy.Payload())
	}

	createdBy := firstNonEmpty(record.GetString("created_by"), responseEnvelope.CreatedBy)
	rc := firstNonEmpty(record.GetString("rc"), responseEnvelope.RC)
	message := firstNonEmpty(record.GetString("message"), responseEnvelope.Message)
	sn := firstNonEmpty(record.GetString("sn"), responseEnvelope.SN)
	statusBefore := firstNonEmpty(proxy.StatusBefore(), responseEnvelope.StatusBefore)
	statusAfter := firstNonEmpty(proxy.StatusAfter(), responseEnvelope.StatusAfter)
	source := firstNonEmpty(proxy.Source(), responseEnvelope.Source)
	redactedPayload := firstNonEmpty(record.GetString("redacted_payload"), proxy.Payload())

	return &DigiflazzEventRecord{
		ID:              proxy.Id,
		FamilyID:        familyID,
		OrderID:         record.GetString("order_id"),
		EventType:       digiflazzdomain.EventType(record.GetString("event_type")),
		StatusBefore:    statusBefore,
		StatusAfter:     statusAfter,
		Source:          source,
		RC:              rc,
		Message:         message,
		SN:              sn,
		RedactedPayload: redactedPayload,
		PayloadHash:     payloadHash,
		ProcessedAt:     processedAtPtr,
		CreatedBy:       createdBy,
		CreatedAt:       proxy.Created().Time(),
		UpdatedAt:       proxy.Updated().Time(),
	}, nil
}

func setDigiflazzEventType(record *core.Record, eventType digiflazzdomain.EventType) error {
	switch eventType {
	case digiflazzdomain.EventTypeTopup:
		record.Set("event_type", "topup")
	case digiflazzdomain.EventTypeInquiry:
		record.Set("event_type", "inquiry")
	case digiflazzdomain.EventTypePay:
		record.Set("event_type", "pay")
	case digiflazzdomain.EventTypeStatus:
		record.Set("event_type", "status")
	case digiflazzdomain.EventTypeDeposit:
		record.Set("event_type", "deposit")
	case digiflazzdomain.EventTypeWebhook:
		record.Set("event_type", "webhook")
	case digiflazzdomain.EventTypeError:
		record.Set("event_type", "error")
	default:
		return fmt.Errorf("unsupported digiflazz event type: %q", eventType)
	}
	return nil
}

func redactDigiflazzPayload(payload string) (string, error) {
	redacted, err := utils.RedactJSON([]byte(payload), digiflazzEventSensitiveFields)
	if err != nil {
		return "", err
	}
	return string(redacted), nil
}

func mustMarshalDigiflazzEventResponse(payload digiflazzEventResponseEnvelope) string {
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeDigiflazzEventResponse(value string) digiflazzEventResponseEnvelope {
	if value == "" {
		return digiflazzEventResponseEnvelope{}
	}

	var envelope digiflazzEventResponseEnvelope
	if err := json.Unmarshal([]byte(value), &envelope); err == nil {
		return envelope
	}

	return digiflazzEventResponseEnvelope{PayloadHash: value}
}

func hashPayload(payload string) string {
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func processedAtOrNow(value *time.Time) types.DateTime {
	if value == nil {
		return types.NowDateTime()
	}
	processedAt, err := types.ParseDateTime(value.UTC())
	if err != nil {
		return types.NowDateTime()
	}
	return processedAt
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func errorsIsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
