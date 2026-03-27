package notifier

import (
	"context"
	"database/sql"
	"testing"

	"github.com/prenansantana/waid/internal/model"
	"github.com/prenansantana/waid/internal/store"
	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := store.RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLiteWebhookStore_CreateAndGet(t *testing.T) {
	db := newTestDB(t)
	s := NewSQLiteWebhookStore(db)
	ctx := context.Background()

	wh := &model.WebhookTarget{
		URL:    "https://example.com/hook",
		Events: []string{model.EventContactResolved, model.EventContactCreated},
		Secret: "mysecret",
	}

	if err := s.CreateWebhook(ctx, wh); err != nil {
		t.Fatalf("CreateWebhook() error: %v", err)
	}
	if wh.ID == "" {
		t.Error("CreateWebhook() did not set ID")
	}

	got, err := s.GetWebhook(ctx, wh.ID)
	if err != nil {
		t.Fatalf("GetWebhook() error: %v", err)
	}
	if got.URL != wh.URL {
		t.Errorf("URL = %q, want %q", got.URL, wh.URL)
	}
	if got.Secret != wh.Secret {
		t.Errorf("Secret = %q, want %q", got.Secret, wh.Secret)
	}
	if len(got.Events) != 2 {
		t.Errorf("Events len = %d, want 2", len(got.Events))
	}
	if !got.Active {
		t.Error("Active should be true after create")
	}
}

func TestSQLiteWebhookStore_List(t *testing.T) {
	db := newTestDB(t)
	s := NewSQLiteWebhookStore(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		wh := &model.WebhookTarget{
			URL:    "https://example.com/hook",
			Events: []string{model.EventContactResolved},
			Secret: "secret",
		}
		if err := s.CreateWebhook(ctx, wh); err != nil {
			t.Fatalf("CreateWebhook() error: %v", err)
		}
	}

	list, err := s.ListWebhooks(ctx)
	if err != nil {
		t.Fatalf("ListWebhooks() error: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("ListWebhooks() returned %d items, want 3", len(list))
	}
}

func TestSQLiteWebhookStore_Delete(t *testing.T) {
	db := newTestDB(t)
	s := NewSQLiteWebhookStore(db)
	ctx := context.Background()

	wh := &model.WebhookTarget{
		URL:    "https://example.com/hook",
		Events: []string{model.EventContactCreated},
		Secret: "secret",
	}
	if err := s.CreateWebhook(ctx, wh); err != nil {
		t.Fatalf("CreateWebhook() error: %v", err)
	}

	if err := s.DeleteWebhook(ctx, wh.ID); err != nil {
		t.Fatalf("DeleteWebhook() error: %v", err)
	}

	// Should not appear in list after delete.
	list, err := s.ListWebhooks(ctx)
	if err != nil {
		t.Fatalf("ListWebhooks() error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("ListWebhooks() returned %d items after delete, want 0", len(list))
	}
}

func TestSQLiteWebhookStore_DeleteNotFound(t *testing.T) {
	db := newTestDB(t)
	s := NewSQLiteWebhookStore(db)
	ctx := context.Background()

	err := s.DeleteWebhook(ctx, "nonexistent-id")
	if err == nil {
		t.Error("DeleteWebhook() should return error for nonexistent ID")
	}
}
