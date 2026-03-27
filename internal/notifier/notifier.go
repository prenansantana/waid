// Package notifier handles outbound webhooks and NATS event publishing.
package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prenansantana/waid/internal/model"
)

// Notifier delivers events to registered webhook targets.
type Notifier struct {
	store  WebhookStore
	client *http.Client
	logger *slog.Logger
}

// NewNotifier creates a Notifier with the given store and logger.
func NewNotifier(store WebhookStore, logger *slog.Logger) *Notifier {
	return &Notifier{
		store:  store,
		client: &http.Client{Timeout: 10 * time.Second},
		logger: logger,
	}
}

// Notify fans out the event to all matching active webhook targets concurrently.
func (n *Notifier) Notify(ctx context.Context, event model.IdentityEvent) error {
	webhooks, err := n.store.ListWebhooks(ctx)
	if err != nil {
		return fmt.Errorf("notifier: list webhooks: %w", err)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("notifier: marshal event: %w", err)
	}

	var wg sync.WaitGroup
	for _, wh := range webhooks {
		if !matchesEvent(wh.Events, event.Type) {
			continue
		}
		wh := wh // capture
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := n.deliver(ctx, wh, payload); err != nil {
				n.logger.Error("notifier: delivery failed",
					"webhook_id", wh.ID,
					"url", wh.URL,
					"error", err,
				)
			}
		}()
	}
	wg.Wait()
	return nil
}

// deliver sends the payload to a single webhook target with exponential backoff retry.
func (n *Notifier) deliver(ctx context.Context, wh model.WebhookTarget, payload []byte) error {
	sig := Sign(payload, wh.Secret)

	delays := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delays[attempt-1]):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("notifier: build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-WAID-Signature", sig)

		resp, err := n.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
			n.logger.Warn("notifier: request failed, will retry",
				"webhook_id", wh.ID,
				"attempt", attempt+1,
				"error", err,
			)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("attempt %d: unexpected status %d", attempt+1, resp.StatusCode)
		n.logger.Warn("notifier: non-2xx response, will retry",
			"webhook_id", wh.ID,
			"attempt", attempt+1,
			"status", resp.StatusCode,
		)
	}
	return fmt.Errorf("notifier: all attempts failed for %s: %w", wh.URL, lastErr)
}

// matchesEvent reports whether eventType matches any of the subscribed events.
// An empty events list matches all event types.
func matchesEvent(events []string, eventType string) bool {
	if len(events) == 0 {
		return true
	}
	for _, e := range events {
		if e == eventType {
			return true
		}
	}
	return false
}
