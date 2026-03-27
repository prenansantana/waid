package adapter

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/prenansantana/waid/internal/model"
	"github.com/prenansantana/waid/pkg/phone"
)

// EvolutionAdapter parses webhooks from the Evolution API.
//
// Expected payload shape:
//
//	{"event": "messages.upsert", "data": {"key": {"remoteJid": "5511999990000@s.whatsapp.net"}, "message": {...}, "pushName": "John"}}
type EvolutionAdapter struct{}

func (a *EvolutionAdapter) Name() string { return "evolution" }

type evolutionPayload struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type evolutionData struct {
	Key      evolutionKey `json:"key"`
	PushName string       `json:"pushName"`
}

type evolutionKey struct {
	RemoteJid string `json:"remoteJid"`
}

func (a *EvolutionAdapter) ParseWebhook(r *http.Request) (*model.InboundEvent, error) {
	defer r.Body.Close()
	var p evolutionPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		return nil, err
	}

	var data evolutionData
	if err := json.Unmarshal(p.Data, &data); err != nil {
		return nil, err
	}
	if data.Key.RemoteJid == "" {
		return nil, errMissingField("data.key.remoteJid")
	}

	stripped := phone.StripJID(data.Key.RemoteJid)
	normalized, err := phone.Normalize(stripped)
	if err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(p)
	return &model.InboundEvent{
		SourceID:    data.Key.RemoteJid,
		Phone:       normalized,
		DisplayName: data.PushName,
		Source:      a.Name(),
		RawPayload:  json.RawMessage(raw),
		Timestamp:   time.Now().UTC(),
	}, nil
}
