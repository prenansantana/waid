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
)

// PostgresWebhookStore is a WebhookStore backed by PostgreSQL.
type PostgresWebhookStore struct {
	db *sql.DB
}

// NewPostgresWebhookStore returns a PostgresWebhookStore using the provided db.
func NewPostgresWebhookStore(db *sql.DB) *PostgresWebhookStore {
	return &PostgresWebhookStore{db: db}
}

// Close closes the underlying database connection.
func (s *PostgresWebhookStore) Close() error {
	return s.db.Close()
}

// CreateWebhook inserts a new WebhookTarget. If wh.ID is empty a new UUID is generated.
func (s *PostgresWebhookStore) CreateWebhook(ctx context.Context, wh *model.WebhookTarget) error {
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
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		wh.ID, wh.URL, string(eventsJSON), wh.Secret, wh.Active, wh.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("notifier: create webhook: %w", err)
	}
	return nil
}

// ListWebhooks returns all active webhook targets.
func (s *PostgresWebhookStore) ListWebhooks(ctx context.Context) ([]model.WebhookTarget, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, url, events, secret, active, created_at FROM webhook_targets WHERE active = true`,
	)
	if err != nil {
		return nil, fmt.Errorf("notifier: list webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []model.WebhookTarget
	for rows.Next() {
		wh, err := scanPgWebhook(rows)
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
func (s *PostgresWebhookStore) GetWebhook(ctx context.Context, id string) (*model.WebhookTarget, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, url, events, secret, active, created_at FROM webhook_targets WHERE id = $1`, id,
	)
	wh, err := scanPgWebhookRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("notifier: webhook %q not found: %w", id, err)
	}
	return wh, err
}

// DeleteWebhook soft-deletes a webhook target by setting active = false.
func (s *PostgresWebhookStore) DeleteWebhook(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE webhook_targets SET active = false WHERE id = $1 AND active = true`, id,
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

func scanPgWebhook(row webhookRowScanner) (*model.WebhookTarget, error) {
	return scanPgWebhookRow(row)
}

func scanPgWebhookRow(row webhookRowScanner) (*model.WebhookTarget, error) {
	var (
		wh         model.WebhookTarget
		eventsJSON string
	)
	if err := row.Scan(&wh.ID, &wh.URL, &eventsJSON, &wh.Secret, &wh.Active, &wh.CreatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(eventsJSON), &wh.Events); err != nil {
		return nil, fmt.Errorf("notifier: unmarshal events: %w", err)
	}
	wh.CreatedAt = wh.CreatedAt.UTC()
	return &wh, nil
}
