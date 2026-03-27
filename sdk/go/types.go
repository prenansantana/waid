package waid

import (
	"encoding/json"
	"time"
)

// Contact represents a resolved WhatsApp identity.
type Contact struct {
	ID         string          `json:"id"`
	Phone      string          `json:"phone"`
	BSUID      *string         `json:"bsuid,omitempty"`
	ExternalID *string         `json:"external_id,omitempty"`
	Name       string          `json:"name"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	Status     string          `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	DeletedAt  *time.Time      `json:"deleted_at,omitempty"`
}

// IdentityResult holds the outcome of an identity resolution attempt.
type IdentityResult struct {
	Contact    *Contact  `json:"contact,omitempty"`
	MatchType  string    `json:"match_type"` // phone, bsuid, created, not_found
	Confidence float64   `json:"confidence"`
	ResolvedAt time.Time `json:"resolved_at"`
}

// ImportError captures a single row error during bulk import.
type ImportError struct {
	Row    int    `json:"row"`
	Phone  string `json:"phone"`
	Reason string `json:"reason"`
}

// ImportReport represents the result of a bulk import.
type ImportReport struct {
	Total   int           `json:"total"`
	Created int           `json:"created"`
	Updated int           `json:"updated"`
	Errors  int           `json:"errors"`
	Details []ImportError `json:"details,omitempty"`
}

// WebhookTarget represents a registered webhook endpoint.
type WebhookTarget struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret,omitempty"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// HealthStatus represents the health check response.
type HealthStatus struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Version  string `json:"version"`
}

// PaginatedContacts is the paginated list response for contacts.
type PaginatedContacts struct {
	Data    []Contact `json:"data"`
	Total   int       `json:"total"`
	Page    int       `json:"page"`
	PerPage int       `json:"per_page"`
}

// CreateContactInput is the request body for creating a contact.
type CreateContactInput struct {
	Phone      string          `json:"phone"`
	Name       string          `json:"name"`
	ExternalID *string         `json:"external_id,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// UpdateContactInput is the request body for updating a contact (all fields optional).
type UpdateContactInput struct {
	Name       *string         `json:"name,omitempty"`
	ExternalID *string         `json:"external_id,omitempty"`
	Status     *string         `json:"status,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// ListOpts holds pagination and search parameters for listing contacts.
type ListOpts struct {
	Page    int
	PerPage int
	Query   string
}

// CreateWebhookInput is the request body for registering a webhook target.
type CreateWebhookInput struct {
	URL    string   `json:"url"`
	Events []string `json:"events,omitempty"`
	Secret string   `json:"secret,omitempty"`
}
