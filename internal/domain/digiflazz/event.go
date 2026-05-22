package digiflazz

import (
	"encoding/json"
	"time"
)

type EventDTO struct {
	ID          string          `json:"id"`
	FamilyID    string          `json:"family_id"`
	OrderID     string          `json:"order_id,omitempty"`
	Type        EventType       `json:"type"`
	Signature   string          `json:"signature,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	ReceivedAt  time.Time       `json:"received_at"`
	CreatedAt   time.Time       `json:"created_at"`
	ProcessedAt *time.Time      `json:"processed_at,omitempty"`
	Error       string          `json:"error,omitempty"`
}

type CreateEventRequest struct {
	FamilyID   string          `json:"family_id"`
	OrderID    string          `json:"order_id,omitempty"`
	Type       EventType       `json:"type"`
	Signature  string          `json:"signature,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	ReceivedAt time.Time       `json:"received_at"`
}
