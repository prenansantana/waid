package adapter

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/prenansantana/waid/internal/model"
	"github.com/prenansantana/waid/pkg/phone"
)

// GenericAdapter parses a simple JSON payload with phone, name, and metadata fields.
//
// Expected payload shape:
//
//	{"phone": "+5511999990000", "name": "John", "whatsapp_id": "...", "metadata": {...}}
type GenericAdapter struct{}

func (a *GenericAdapter) Name() string { return "generic" }

type genericPayload struct {
	Phone      string          `json:"phone"`
	Name       string          `json:"name"`
	WhatsAppID string          `json:"whatsapp_id,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

func (a *GenericAdapter) ParseWebhook(r *http.Request) (*model.InboundEvent, error) {
	defer r.Body.Close()
	var p genericPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		return nil, err
	}
	if p.Phone == "" {
		return nil, errMissingField("phone")
	}

	normalized, err := phone.Normalize(p.Phone, "")
	if err != nil {
		return nil, err
	}

	var whatsAppID *string
	if p.WhatsAppID != "" {
		whatsAppID = &p.WhatsAppID
	}

	raw, _ := json.Marshal(p)
	return &model.InboundEvent{
		SourceID:    p.Phone,
		Phone:       normalized,
		DisplayName: p.Name,
		WhatsAppID:  whatsAppID,
		Source:      a.Name(),
		RawPayload:  json.RawMessage(raw),
		Timestamp:   time.Now().UTC(),
	}, nil
}
