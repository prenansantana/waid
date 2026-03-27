package adapter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prenansantana/waid/internal/model"
	"github.com/prenansantana/waid/pkg/phone"
)

// MetaAdapter parses webhooks from the Meta Cloud API (WhatsApp Business).
//
// Expected payload shape:
//
//	{"object": "whatsapp_business_account", "entry": [{"changes": [{"value": {"messages": [...], "contacts": [...]}}]}]}
type MetaAdapter struct{}

func (a *MetaAdapter) Name() string { return "meta" }

type metaPayload struct {
	Object string      `json:"object"`
	Entry  []metaEntry `json:"entry"`
}

type metaEntry struct {
	Changes []metaChange `json:"changes"`
}

type metaChange struct {
	Value metaValue `json:"value"`
}

type metaValue struct {
	Messages []metaMessage `json:"messages"`
	Contacts []metaContact `json:"contacts"`
}

type metaMessage struct {
	From string `json:"from"`
	Type string `json:"type"`
}

type metaContact struct {
	WaID    string             `json:"wa_id"`
	Profile metaContactProfile `json:"profile"`
}

type metaContactProfile struct {
	Name string `json:"name"`
}

func (a *MetaAdapter) ParseWebhook(r *http.Request) (*model.InboundEvent, error) {
	defer r.Body.Close()
	var p metaPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		return nil, err
	}

	if len(p.Entry) == 0 {
		return nil, errMissingField("entry")
	}
	if len(p.Entry[0].Changes) == 0 {
		return nil, errMissingField("entry[0].changes")
	}

	value := p.Entry[0].Changes[0].Value

	if len(value.Messages) == 0 {
		return nil, errMissingField("entry[0].changes[0].value.messages")
	}

	msg := value.Messages[0]
	if msg.From == "" {
		return nil, errMissingField("messages[0].from")
	}

	normalized, err := phone.Normalize("+"+msg.From, "")
	if err != nil {
		return nil, err
	}

	var displayName string
	var whatsAppID *string
	if len(value.Contacts) > 0 {
		displayName = value.Contacts[0].Profile.Name
		waID := value.Contacts[0].WaID
		if waID != "" {
			whatsAppID = &waID
		}
	}

	raw, _ := json.Marshal(p)
	return &model.InboundEvent{
		SourceID:    msg.From,
		Phone:       normalized,
		DisplayName: displayName,
		WhatsAppID:  whatsAppID,
		Source:      a.Name(),
		RawPayload:  json.RawMessage(raw),
		Timestamp:   time.Now().UTC(),
	}, nil
}

// VerifyWebhook handles the Meta webhook verification handshake (GET request).
// It checks hub.mode == "subscribe" and hub.verify_token matches, then returns hub.challenge.
func (a *MetaAdapter) VerifyWebhook(r *http.Request, verifyToken string) (string, error) {
	q := r.URL.Query()
	if q.Get("hub.mode") != "subscribe" {
		return "", fmt.Errorf("adapter: invalid hub.mode %q", q.Get("hub.mode"))
	}
	if q.Get("hub.verify_token") != verifyToken {
		return "", fmt.Errorf("adapter: verify_token mismatch")
	}
	challenge := q.Get("hub.challenge")
	if challenge == "" {
		return "", errMissingField("hub.challenge")
	}
	return challenge, nil
}
