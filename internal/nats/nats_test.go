package nats_test

import (
	"sync"
	"testing"
	"time"

	natsgo "github.com/nats-io/nats.go"

	"github.com/prenansantana/waid/internal/config"
	"github.com/prenansantana/waid/internal/nats"
)

func newEmbedded(t *testing.T) *nats.NATS {
	t.Helper()
	n, err := nats.NewNATS(config.NATSConfig{Embedded: true, URL: ""}, testLogger(t))
	if err != nil {
		t.Fatalf("NewNATS embedded: %v", err)
	}
	t.Cleanup(func() { _ = n.Close() })
	return n
}

func TestEmbedded_StartPublishSubscribeReceiveClose(t *testing.T) {
	n := newEmbedded(t)

	subject := "waid.inbound.test"
	payload := []byte("hello-waid")

	var (
		mu      sync.Mutex
		got     []byte
		received = make(chan struct{})
	)

	_, err := n.Subscribe(subject, func(msg *natsgo.Msg) {
		mu.Lock()
		got = msg.Data
		mu.Unlock()
		close(received)
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	if err := n.Publish(subject, payload); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	mu.Lock()
	defer mu.Unlock()
	if string(got) != string(payload) {
		t.Errorf("got %q, want %q", got, payload)
	}
}

func TestPing_Connected(t *testing.T) {
	n := newEmbedded(t)
	if err := n.Ping(); err != nil {
		t.Errorf("Ping on connected NATS: %v", err)
	}
}

func TestPing_AfterClose(t *testing.T) {
	n, err := nats.NewNATS(config.NATSConfig{Embedded: true, URL: ""}, testLogger(t))
	if err != nil {
		t.Fatalf("NewNATS: %v", err)
	}

	if err := n.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := n.Ping(); err == nil {
		t.Error("expected Ping to fail after Close, got nil")
	}
}

func TestWildcardSubject(t *testing.T) {
	n := newEmbedded(t)

	subjects := []string{
		"waid.inbound.whatsapp",
		"waid.identity.created",
	}

	received := make(chan string, len(subjects))

	_, err := n.Subscribe("waid.>", func(msg *natsgo.Msg) {
		received <- msg.Subject
	})
	if err != nil {
		t.Fatalf("Subscribe wildcard: %v", err)
	}

	for _, s := range subjects {
		if err := n.Publish(s, []byte("data")); err != nil {
			t.Fatalf("Publish %s: %v", s, err)
		}
	}

	deadline := time.After(3 * time.Second)
	seen := make(map[string]bool)
	for i := 0; i < len(subjects); i++ {
		select {
		case s := <-received:
			seen[s] = true
		case <-deadline:
			t.Fatalf("timed out; received %d/%d messages", i, len(subjects))
		}
	}

	for _, s := range subjects {
		if !seen[s] {
			t.Errorf("did not receive message on subject %s", s)
		}
	}
}
