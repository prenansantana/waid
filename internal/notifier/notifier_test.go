package notifier

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prenansantana/waid/internal/model"
)

// mockWebhookStore implements WebhookStore for testing.
type mockWebhookStore struct {
	webhooks []model.WebhookTarget
}

func (m *mockWebhookStore) CreateWebhook(_ context.Context, wh *model.WebhookTarget) error {
	m.webhooks = append(m.webhooks, *wh)
	return nil
}

func (m *mockWebhookStore) ListWebhooks(_ context.Context) ([]model.WebhookTarget, error) {
	return m.webhooks, nil
}

func (m *mockWebhookStore) GetWebhook(_ context.Context, id string) (*model.WebhookTarget, error) {
	for _, wh := range m.webhooks {
		if wh.ID == id {
			return &wh, nil
		}
	}
	return nil, nil
}

func (m *mockWebhookStore) DeleteWebhook(_ context.Context, id string) error {
	for i, wh := range m.webhooks {
		if wh.ID == id {
			m.webhooks = append(m.webhooks[:i], m.webhooks[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockWebhookStore) Close() error { return nil }

func TestNotify_DeliveryPayloadAndSignature(t *testing.T) {
	secret := "testsecret"
	event := model.IdentityEvent{
		Type:      model.EventContactResolved,
		Phone:     "+5511999999999",
		Source:    "whatsapp",
		Timestamp: time.Now().UTC(),
	}

	var receivedBody []byte
	var receivedSig string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-WAID-Signature")
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := &mockWebhookStore{
		webhooks: []model.WebhookTarget{
			{ID: "wh1", URL: srv.URL, Events: []string{model.EventContactResolved}, Secret: secret, Active: true},
		},
	}

	n := NewNotifier(store, slog.Default())
	if err := n.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify() error: %v", err)
	}

	// Verify payload is valid JSON matching the event.
	var decoded model.IdentityEvent
	if err := json.Unmarshal(receivedBody, &decoded); err != nil {
		t.Fatalf("received body is not valid JSON: %v", err)
	}
	if decoded.Type != event.Type {
		t.Errorf("decoded event type = %q, want %q", decoded.Type, event.Type)
	}

	// Verify HMAC signature.
	if !Verify(receivedBody, secret, receivedSig) {
		t.Errorf("signature verification failed: sig=%q", receivedSig)
	}
}

func TestNotify_EventFiltering(t *testing.T) {
	var hitCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := &mockWebhookStore{
		webhooks: []model.WebhookTarget{
			{ID: "wh1", URL: srv.URL, Events: []string{model.EventContactCreated}, Active: true},
			{ID: "wh2", URL: srv.URL, Events: []string{model.EventContactResolved}, Active: true},
			{ID: "wh3", URL: srv.URL, Events: []string{}, Active: true}, // matches all
		},
	}

	n := NewNotifier(store, slog.Default())
	event := model.IdentityEvent{Type: model.EventContactCreated, Phone: "+1234567890", Timestamp: time.Now()}
	if err := n.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify() error: %v", err)
	}

	// wh1 (contact.created) + wh3 (all) = 2 deliveries
	if got := hitCount.Load(); got != 2 {
		t.Errorf("expected 2 webhook deliveries, got %d", got)
	}
}

func TestNotify_RetryBehavior(t *testing.T) {
	var attempts atomic.Int32
	done := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		close(done)
	}))
	defer srv.Close()

	ms := &mockWebhookStore{
		webhooks: []model.WebhookTarget{
			{ID: "wh1", URL: srv.URL, Events: []string{model.EventContactResolved}, Active: true},
		},
	}

	n := NewNotifier(ms, slog.Default())
	n.client = &http.Client{Timeout: 5 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	event := model.IdentityEvent{Type: model.EventContactResolved, Phone: "+1234567890", Timestamp: time.Now()}
	if err := n.Notify(ctx, event); err != nil {
		t.Fatalf("Notify() unexpected error: %v", err)
	}

	// Wait for success signal or timeout.
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for successful delivery after retries")
	}

	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 delivery attempts (retry), got %d", got)
	}
}

func TestNotify_AllRetriesFail(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	ms := &mockWebhookStore{
		webhooks: []model.WebhookTarget{
			{ID: "wh1", URL: srv.URL, Events: []string{model.EventContactResolved}, Active: true},
		},
	}

	n := NewNotifier(ms, slog.Default())
	n.client = &http.Client{Timeout: 5 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	event := model.IdentityEvent{Type: model.EventContactResolved, Phone: "+1234567890", Timestamp: time.Now()}
	if err := n.Notify(ctx, event); err != nil {
		t.Fatalf("Notify() unexpected error: %v", err)
	}

	// Wait for all retry goroutines to finish (1s + 2s delays + overhead).
	time.Sleep(8 * time.Second)

	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 delivery attempts, got %d", got)
	}
}
