package notifier

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/prenansantana/waid/internal/model"
	_ "modernc.org/sqlite"
)

// WebhookStore is the persistence interface for webhook targets.
type WebhookStore interface {
	CreateWebhook(ctx context.Context, wh *model.WebhookTarget) error
	ListWebhooks(ctx context.Context) ([]model.WebhookTarget, error)
	GetWebhook(ctx context.Context, id string) (*model.WebhookTarget, error)
	DeleteWebhook(ctx context.Context, id string) error
}

// SQLiteWebhookStore is a WebhookStore backed by SQLite.
type SQLiteWebhookStore struct {
	db *sql.DB
}

// NewSQLiteWebhookStore returns a SQLiteWebhookStore using the provided db.
func NewSQLiteWebhookStore(db *sql.DB) *SQLiteWebhookStore {
	return &SQLiteWebhookStore{db: db}
}

// CreateWebhook inserts a new WebhookTarget. If wh.ID is empty a new UUID is generated.
func (s *SQLiteWebhookStore) CreateWebhook(ctx context.Context, wh *model.WebhookTarget) error {
	if wh.ID == "" {
		wh.ID = uuid.New().String()
	}
	wh.CreatedAt = time.Now().UTC()
	wh.Active = true

	eventsJSON, err := json.Marshal(wh.Events)
	if err != nil {
		return fmt.Errorf("notifier: marshal events: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO webhook_targets (id, url, events, secret, active, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		wh.ID, wh.URL, string(eventsJSON), wh.Secret, wh.Active,
		wh.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("notifier: create webhook: %w", err)
	}
	return nil
}

// ListWebhooks returns all active webhook targets.
func (s *SQLiteWebhookStore) ListWebhooks(ctx context.Context) ([]model.WebhookTarget, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, url, events, secret, active, created_at FROM webhook_targets WHERE active = true`,
	)
	if err != nil {
		return nil, fmt.Errorf("notifier: list webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []model.WebhookTarget
	for rows.Next() {
		wh, err := scanWebhook(rows)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, *wh)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("notifier: list webhooks rows: %w", err)
	}
	return webhooks, nil
}

// GetWebhook returns a single webhook target by ID.
func (s *SQLiteWebhookStore) GetWebhook(ctx context.Context, id string) (*model.WebhookTarget, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, url, events, secret, active, created_at FROM webhook_targets WHERE id = ?`, id,
	)
	wh, err := scanWebhookRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("notifier: webhook %q not found: %w", id, err)
	}
	return wh, err
}

// DeleteWebhook soft-deletes a webhook target by setting active = false.
func (s *SQLiteWebhookStore) DeleteWebhook(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE webhook_targets SET active = false WHERE id = ? AND active = true`, id,
	)
	if err != nil {
		return fmt.Errorf("notifier: delete webhook: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("notifier: webhook %q not found", id)
	}
	return nil
}

type webhookRowScanner interface {
	Scan(dest ...any) error
}

func scanWebhook(row webhookRowScanner) (*model.WebhookTarget, error) {
	return scanWebhookRow(row)
}

func scanWebhookRow(row webhookRowScanner) (*model.WebhookTarget, error) {
	var (
		wh         model.WebhookTarget
		eventsJSON string
		createdAt  string
	)
	if err := row.Scan(&wh.ID, &wh.URL, &eventsJSON, &wh.Secret, &wh.Active, &createdAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(eventsJSON), &wh.Events); err != nil {
		return nil, fmt.Errorf("notifier: unmarshal events: %w", err)
	}
	t, err := parseWebhookTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("notifier: parse created_at: %w", err)
	}
	wh.CreatedAt = t
	return &wh, nil
}

func parseWebhookTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999999",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised time format: %q", s)
}
