package adapter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prenansantana/waid/internal/model"
	"github.com/prenansantana/waid/pkg/phone"
)

// WAHAAdapter parses webhooks from WAHA (WhatsApp HTTP API).
//
// Expected payload shape:
//
//	{"event": "message", "payload": {"from": "5511999990000@c.us", "body": "text", ...}}
type WAHAAdapter struct{}

func (a *WAHAAdapter) Name() string { return "waha" }

type wahaPayload struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

type wahaInner struct {
	From string `json:"from"`
}

func (a *WAHAAdapter) ParseWebhook(r *http.Request) (*model.InboundEvent, error) {
	defer r.Body.Close()
	var p wahaPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		return nil, err
	}

	var inner wahaInner
	if err := json.Unmarshal(p.Payload, &inner); err != nil {
		return nil, err
	}
	if inner.From == "" {
		return nil, errMissingField("payload.from")
	}

	// Filter group messages.
	if strings.HasSuffix(inner.From, "@g.us") {
		return nil, fmt.Errorf("adapter: group messages not supported")
	}

	// Preserve original JID as WhatsAppID before stripping.
	whatsAppID := inner.From

	stripped := phone.StripJID(inner.From)
	normalized, err := phone.Normalize("+"+stripped, "")
	if err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(p)
	return &model.InboundEvent{
		SourceID:   inner.From,
		Phone:      normalized,
		WhatsAppID: &whatsAppID,
		Source:     a.Name(),
		RawPayload: json.RawMessage(raw),
		Timestamp:  time.Now().UTC(),
	}, nil
}
