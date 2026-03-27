package model

import (
	"encoding/json"
	"time"
)

// Identity event type constants.
const (
	EventContactResolved = "contact.resolved"
	EventContactCreated  = "contact.created"
	EventContactUpdated  = "contact.updated"
	EventContactNotFound = "contact.not_found"
)

// IdentityEvent is published to NATS/webhooks on identity resolution outcomes.
type IdentityEvent struct {
	Type      string          `json:"type"`
	Contact   *Contact        `json:"contact,omitempty"`
	Phone     string          `json:"phone"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}
