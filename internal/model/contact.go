package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Contact represents a resolved WhatsApp identity.
type Contact struct {
	ID          string          `json:"id"`
	Phone       string          `json:"phone"`
	BSUID       *string         `json:"bsuid,omitempty"`
	ExternalID  *string         `json:"external_id,omitempty"`
	WhatsAppID  *string         `json:"whatsapp_id,omitempty"`
	Name        string          `json:"name"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty"`
}

// InboundEvent represents a raw webhook event received from WhatsApp.
type InboundEvent struct {
	SourceID    string          `json:"source_id"`
	Phone       string          `json:"phone"`
	BSUID       *string         `json:"bsuid,omitempty"`
	WhatsAppID  *string         `json:"whatsapp_id,omitempty"`
	DisplayName string          `json:"display_name,omitempty"`
	Source      string          `json:"source"`
	RawPayload  json.RawMessage `json:"raw_payload,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
}

// IdentityResult holds the outcome of an identity resolution attempt.
type IdentityResult struct {
	Contact    *Contact  `json:"contact,omitempty"`
	MatchType  string    `json:"match_type"` // phone, bsuid, created, not_found
	Confidence float64   `json:"confidence"`
	ResolvedAt time.Time `json:"resolved_at"`
}

// WebhookTarget represents a registered webhook endpoint.
type WebhookTarget struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"` // contact.resolved, contact.created, etc.
	Secret    string    `json:"secret,omitempty"` // for HMAC signing
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// ImportReport represents the result of a bulk import.
type ImportReport struct {
	Total   int           `json:"total"`
	Created int           `json:"created"`
	Updated int           `json:"updated"`
	Errors  int           `json:"errors"`
	Details []ImportError `json:"details,omitempty"`
}

// ImportError captures a single row error during bulk import.
type ImportError struct {
	Row    int    `json:"row"`
	Phone  string `json:"phone"`
	Reason string `json:"reason"`
}

// NewContact creates a Contact with a new UUID and current timestamps.
func NewContact(phone, name string) *Contact {
	now := time.Now().UTC()
	return &Contact{
		ID:        uuid.New().String(),
		Phone:     phone,
		Name:      name,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}
}
