package adapter_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prenansantana/waid/internal/adapter"
)

// --- Registry tests ---

func TestRegistry(t *testing.T) {
	reg := adapter.NewRegistry()
	reg.Register(&adapter.WAHAAdapter{})

	a, ok := reg.Get("waha")
	if !ok {
		t.Fatal("expected to find waha adapter")
	}
	if a.Name() != "waha" {
		t.Fatalf("expected waha, got %s", a.Name())
	}

	_, ok = reg.Get("missing")
	if ok {
		t.Fatal("expected not to find missing adapter")
	}
}

func TestDefaultRegistry(t *testing.T) {
	reg := adapter.DefaultRegistry()
	for _, name := range []string{"waha", "evolution", "meta", "generic"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("expected adapter %q to be registered", name)
		}
	}
}

// --- WAHA adapter tests ---

func TestWAHAAdapter_ParseWebhook(t *testing.T) {
	payload := `{"event":"message","payload":{"from":"5511999990000@c.us","body":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))

	a := &adapter.WAHAAdapter{}
	event, err := a.ParseWebhook(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Phone != "+5511999990000" {
		t.Fatalf("expected +5511999990000, got %s", event.Phone)
	}
	if event.Source != "waha" {
		t.Fatalf("expected source waha, got %s", event.Source)
	}
	if event.SourceID != "5511999990000@c.us" {
		t.Fatalf("expected SourceID 5511999990000@c.us, got %s", event.SourceID)
	}
}

func TestWAHAAdapter_MissingFrom(t *testing.T) {
	payload := `{"event":"message","payload":{"body":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))

	a := &adapter.WAHAAdapter{}
	_, err := a.ParseWebhook(req)
	if err == nil {
		t.Fatal("expected error for missing from field")
	}
}

func TestWAHAAdapter_MalformedPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`not-json`))
	a := &adapter.WAHAAdapter{}
	_, err := a.ParseWebhook(req)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

// --- Evolution adapter tests ---

func TestEvolutionAdapter_ParseWebhook(t *testing.T) {
	payload := `{"event":"messages.upsert","data":{"key":{"remoteJid":"5511999990000@s.whatsapp.net"},"pushName":"John","message":{}}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))

	a := &adapter.EvolutionAdapter{}
	event, err := a.ParseWebhook(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Phone != "+5511999990000" {
		t.Fatalf("expected +5511999990000, got %s", event.Phone)
	}
	if event.DisplayName != "John" {
		t.Fatalf("expected displayName John, got %s", event.DisplayName)
	}
	if event.Source != "evolution" {
		t.Fatalf("expected source evolution, got %s", event.Source)
	}
}

func TestEvolutionAdapter_MissingRemoteJid(t *testing.T) {
	payload := `{"event":"messages.upsert","data":{"key":{},"pushName":"John"}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))

	a := &adapter.EvolutionAdapter{}
	_, err := a.ParseWebhook(req)
	if err == nil {
		t.Fatal("expected error for missing remoteJid")
	}
}

func TestEvolutionAdapter_MalformedPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad json`))
	a := &adapter.EvolutionAdapter{}
	_, err := a.ParseWebhook(req)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

// --- Meta adapter tests ---

func TestMetaAdapter_ParseWebhook(t *testing.T) {
	payload := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"changes": [{
				"value": {
					"messages": [{"from": "5511999990000", "type": "text"}],
					"contacts": [{"wa_id": "5511999990000", "profile": {"name": "John"}}]
				}
			}]
		}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))

	a := &adapter.MetaAdapter{}
	event, err := a.ParseWebhook(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Phone != "+5511999990000" {
		t.Fatalf("expected +5511999990000, got %s", event.Phone)
	}
	if event.DisplayName != "John" {
		t.Fatalf("expected displayName John, got %s", event.DisplayName)
	}
	if event.Source != "meta" {
		t.Fatalf("expected source meta, got %s", event.Source)
	}
}

func TestMetaAdapter_MissingEntry(t *testing.T) {
	payload := `{"object":"whatsapp_business_account","entry":[]}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))

	a := &adapter.MetaAdapter{}
	_, err := a.ParseWebhook(req)
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestMetaAdapter_MalformedPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`<<<`))
	a := &adapter.MetaAdapter{}
	_, err := a.ParseWebhook(req)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestMetaAdapter_VerifyWebhook(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?hub.mode=subscribe&hub.verify_token=secret&hub.challenge=abc123", nil)

	a := &adapter.MetaAdapter{}
	challenge, err := a.VerifyWebhook(req, "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if challenge != "abc123" {
		t.Fatalf("expected challenge abc123, got %s", challenge)
	}
}

func TestMetaAdapter_VerifyWebhook_WrongToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=abc123", nil)

	a := &adapter.MetaAdapter{}
	_, err := a.VerifyWebhook(req, "secret")
	if err == nil {
		t.Fatal("expected error for wrong verify_token")
	}
}

func TestMetaAdapter_VerifyWebhook_WrongMode(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?hub.mode=unsubscribe&hub.verify_token=secret&hub.challenge=abc123", nil)

	a := &adapter.MetaAdapter{}
	_, err := a.VerifyWebhook(req, "secret")
	if err == nil {
		t.Fatal("expected error for wrong hub.mode")
	}
}

// --- Generic adapter tests ---

func TestGenericAdapter_ParseWebhook(t *testing.T) {
	payload := `{"phone":"+5511999990000","name":"John","metadata":{"key":"value"}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))

	a := &adapter.GenericAdapter{}
	event, err := a.ParseWebhook(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Phone != "+5511999990000" {
		t.Fatalf("expected +5511999990000, got %s", event.Phone)
	}
	if event.DisplayName != "John" {
		t.Fatalf("expected displayName John, got %s", event.DisplayName)
	}
	if event.Source != "generic" {
		t.Fatalf("expected source generic, got %s", event.Source)
	}
}

func TestGenericAdapter_MissingPhone(t *testing.T) {
	payload := `{"name":"John"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))

	a := &adapter.GenericAdapter{}
	_, err := a.ParseWebhook(req)
	if err == nil {
		t.Fatal("expected error for missing phone")
	}
}

func TestGenericAdapter_MalformedPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`))
	a := &adapter.GenericAdapter{}
	_, err := a.ParseWebhook(req)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
