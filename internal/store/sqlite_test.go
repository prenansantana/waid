package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/prenansantana/waid/internal/model"
	"github.com/prenansantana/waid/internal/store"
)

func newTestStore(t *testing.T) store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.NewSQLite(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestPing(t *testing.T) {
	s := newTestStore(t)
	if err := s.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestCreate(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	c := &model.Contact{
		Phone:  "+5511999990001",
		Name:   "Alice",
		Status: "active",
	}
	if err := s.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ID == "" {
		t.Error("expected ID to be set")
	}
	if c.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestFindByPhone(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	c := &model.Contact{Phone: "+5511999990002", Name: "Bob", Status: "active"}
	if err := s.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.FindByPhone(ctx, "+5511999990002")
	if err != nil {
		t.Fatalf("FindByPhone: %v", err)
	}
	if got.ID != c.ID {
		t.Errorf("got ID %q, want %q", got.ID, c.ID)
	}
	if got.Name != "Bob" {
		t.Errorf("got Name %q, want Bob", got.Name)
	}
}

func TestFindByBSUID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	bsuid := "BSUID-ABC-001"
	c := &model.Contact{Phone: "+5511999990003", Name: "Carol", Status: "active", BSUID: &bsuid}
	if err := s.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.FindByBSUID(ctx, bsuid)
	if err != nil {
		t.Fatalf("FindByBSUID: %v", err)
	}
	if got.ID != c.ID {
		t.Errorf("got ID %q, want %q", got.ID, c.ID)
	}
}

func TestFindByID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	c := &model.Contact{Phone: "+5511999990004", Name: "Dave", Status: "active"}
	if err := s.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.FindByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Phone != c.Phone {
		t.Errorf("got Phone %q, want %q", got.Phone, c.Phone)
	}
}

func TestUpdate(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	c := &model.Contact{Phone: "+5511999990005", Name: "Eve", Status: "active"}
	if err := s.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	c.Name = "Eve Updated"
	c.Status = "inactive"
	if err := s.Update(ctx, c); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := s.FindByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("FindByID after update: %v", err)
	}
	if got.Name != "Eve Updated" {
		t.Errorf("got Name %q, want Eve Updated", got.Name)
	}
	if got.Status != "inactive" {
		t.Errorf("got Status %q, want inactive", got.Status)
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	c := &model.Contact{Phone: "+5511999990006", Name: "Frank", Status: "active"}
	if err := s.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.Delete(ctx, c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Soft-deleted — FindByPhone should not return it.
	got, err := s.FindByPhone(ctx, "+5511999990006")
	if err == nil {
		t.Errorf("expected error after delete, got contact %v", got)
	}

	// Double-delete should fail.
	if err := s.Delete(ctx, c.ID); err == nil {
		t.Error("expected error on double-delete")
	}
}

func TestList(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	names := []string{"Grace", "Hank", "Iris"}
	phones := []string{"+5511999990010", "+5511999990011", "+5511999990012"}
	for i, name := range names {
		c := &model.Contact{Phone: phones[i], Name: name, Status: "active"}
		if err := s.Create(ctx, c); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	t.Run("all", func(t *testing.T) {
		contacts, total, err := s.List(ctx, store.ListOpts{Page: 1, PerPage: 10})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 3 {
			t.Errorf("total = %d, want 3", total)
		}
		if len(contacts) != 3 {
			t.Errorf("len(contacts) = %d, want 3", len(contacts))
		}
	})

	t.Run("search by name", func(t *testing.T) {
		contacts, total, err := s.List(ctx, store.ListOpts{Page: 1, PerPage: 10, Query: "Grace"})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 1 {
			t.Errorf("total = %d, want 1", total)
		}
		if len(contacts) != 1 || contacts[0].Name != "Grace" {
			t.Errorf("unexpected contacts: %v", contacts)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		contacts, total, err := s.List(ctx, store.ListOpts{Page: 1, PerPage: 2})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 3 {
			t.Errorf("total = %d, want 3", total)
		}
		if len(contacts) != 2 {
			t.Errorf("len(contacts) = %d, want 2", len(contacts))
		}

		contacts2, _, err := s.List(ctx, store.ListOpts{Page: 2, PerPage: 2})
		if err != nil {
			t.Fatalf("List page 2: %v", err)
		}
		if len(contacts2) != 1 {
			t.Errorf("page 2 len = %d, want 1", len(contacts2))
		}
	})
}

func TestBulkUpsert(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	contacts := []model.Contact{
		{Phone: "+5511999990020", Name: "Jack", Status: "active"},
		{Phone: "+5511999990021", Name: "Kate", Status: "active"},
		{Phone: "+5511999990022", Name: "Leo", Status: "active"},
	}

	report, err := s.BulkUpsert(ctx, contacts)
	if err != nil {
		t.Fatalf("BulkUpsert: %v", err)
	}
	if report.Total != 3 {
		t.Errorf("Total = %d, want 3", report.Total)
	}
	if report.Created != 3 {
		t.Errorf("Created = %d, want 3", report.Created)
	}
	if report.Errors != 0 {
		t.Errorf("Errors = %d, want 0", report.Errors)
	}

	// Upsert same phones — should update.
	contacts[0].Name = "Jack Updated"
	report2, err := s.BulkUpsert(ctx, contacts[:1])
	if err != nil {
		t.Fatalf("BulkUpsert (update): %v", err)
	}
	if report2.Updated != 1 {
		t.Errorf("Updated = %d, want 1", report2.Updated)
	}

	got, err := s.FindByPhone(ctx, "+5511999990020")
	if err != nil {
		t.Fatalf("FindByPhone after upsert: %v", err)
	}
	if got.Name != "Jack Updated" {
		t.Errorf("Name = %q, want Jack Updated", got.Name)
	}
}

func TestFactory(t *testing.T) {
	dir := t.TempDir()
	s, err := store.New("sqlite", filepath.Join(dir, "factory.db"))
	if err != nil {
		t.Fatalf("store.New sqlite: %v", err)
	}
	defer s.Close()

	if err := s.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestFactoryUnsupported(t *testing.T) {
	_, err := store.New("mysql", "dsn")
	if err == nil {
		t.Error("expected error for unsupported driver")
	}
}

// Ensure test binary can run even if no TMPDIR is set.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
