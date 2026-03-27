package resolver_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/prenansantana/waid/internal/model"
	"github.com/prenansantana/waid/internal/resolver"
	"github.com/prenansantana/waid/internal/store"
)

// mockStore is an in-memory implementation of store.Store for testing.
type mockStore struct {
	byPhone map[string]*model.Contact
	byBSUID map[string]*model.Contact
	byID    map[string]*model.Contact
}

func newMockStore() *mockStore {
	return &mockStore{
		byPhone: make(map[string]*model.Contact),
		byBSUID: make(map[string]*model.Contact),
		byID:    make(map[string]*model.Contact),
	}
}

func (m *mockStore) add(c *model.Contact) {
	m.byID[c.ID] = c
	m.byPhone[c.Phone] = c
	if c.BSUID != nil {
		m.byBSUID[*c.BSUID] = c
	}
}

func (m *mockStore) FindByPhone(_ context.Context, phone string) (*model.Contact, error) {
	c, ok := m.byPhone[phone]
	if !ok {
		return nil, store.ErrNotFound
	}
	return c, nil
}

func (m *mockStore) FindByBSUID(_ context.Context, id string) (*model.Contact, error) {
	c, ok := m.byBSUID[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return c, nil
}

func (m *mockStore) FindByExternalID(_ context.Context, _ string) (*model.Contact, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) FindByWhatsAppID(_ context.Context, _ string) (*model.Contact, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) FindByID(_ context.Context, id string) (*model.Contact, error) {
	c, ok := m.byID[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return c, nil
}

func (m *mockStore) Create(_ context.Context, c *model.Contact) error {
	m.add(c)
	return nil
}

func (m *mockStore) Update(_ context.Context, c *model.Contact) error {
	old, ok := m.byID[c.ID]
	if !ok {
		return errors.New("not found")
	}
	// Remove old phone/bsuid index entries.
	delete(m.byPhone, old.Phone)
	if old.BSUID != nil {
		delete(m.byBSUID, *old.BSUID)
	}
	m.add(c)
	return nil
}

func (m *mockStore) Delete(_ context.Context, id string) error {
	c, ok := m.byID[id]
	if !ok {
		return errors.New("not found")
	}
	now := time.Now().UTC()
	c.DeletedAt = &now
	return nil
}

func (m *mockStore) List(_ context.Context, _ store.ListOpts) ([]model.Contact, int, error) {
	var out []model.Contact
	for _, c := range m.byID {
		out = append(out, *c)
	}
	return out, len(out), nil
}

func (m *mockStore) BulkUpsert(_ context.Context, contacts []model.Contact) (*model.ImportReport, error) {
	rep := &model.ImportReport{Total: len(contacts)}
	for i := range contacts {
		m.add(&contacts[i])
		rep.Created++
	}
	return rep, nil
}

func (m *mockStore) Ping(_ context.Context) error { return nil }
func (m *mockStore) Close() error                  { return nil }

// helpers

func strPtr(s string) *string { return &s }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// Tests — phone numbers stored in E.164 format (as produced by phone.Normalize).

func TestResolve_BSUIDMatch(t *testing.T) {
	s := newMockStore()
	bsuID := "BR.alice12345678" // valid BSUID: CC.alphanumericID
	c := model.NewContact("+5511999990000", "Alice")
	c.BSUID = strPtr(bsuID)
	s.add(c)

	r := resolver.New(s, false, "BR", testLogger())
	evt := model.InboundEvent{
		Phone:     "+55 11 99999-0000",
		BSUID:     strPtr(bsuID),
		Timestamp: time.Now(),
	}
	result, err := r.Resolve(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != "bsuid" {
		t.Errorf("expected match_type=bsuid, got %q", result.MatchType)
	}
	if result.Confidence != 1.0 {
		t.Errorf("expected confidence=1.0, got %f", result.Confidence)
	}
	if result.Contact == nil || result.Contact.ID != c.ID {
		t.Errorf("expected contact %s, got %v", c.ID, result.Contact)
	}
}

func TestResolve_PhoneMatch(t *testing.T) {
	s := newMockStore()
	c := model.NewContact("+5511999990001", "Bob")
	s.add(c)

	r := resolver.New(s, false, "BR", testLogger())
	evt := model.InboundEvent{
		Phone:     "+55 (11) 99999-0001",
		Timestamp: time.Now(),
	}
	result, err := r.Resolve(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != "phone" {
		t.Errorf("expected match_type=phone, got %q", result.MatchType)
	}
	if result.Contact == nil || result.Contact.ID != c.ID {
		t.Errorf("expected contact %s", c.ID)
	}
}

func TestResolve_BSUIDBackfill(t *testing.T) {
	s := newMockStore()
	c := model.NewContact("+5511999990002", "Carol")
	// No BSUID initially.
	s.add(c)

	bsuID := "BR.carol12345678" // valid BSUID
	r := resolver.New(s, false, "BR", testLogger())
	evt := model.InboundEvent{
		Phone:     "+5511999990002",
		BSUID:     strPtr(bsuID),
		Timestamp: time.Now(),
	}
	result, err := r.Resolve(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != "phone" {
		t.Errorf("expected match_type=phone, got %q", result.MatchType)
	}
	// BSUID should have been backfilled.
	if result.Contact.BSUID == nil || *result.Contact.BSUID != bsuID {
		t.Errorf("expected bsuid %s to be backfilled, got %v", bsuID, result.Contact.BSUID)
	}
}

func TestResolve_AutoCreate(t *testing.T) {
	s := newMockStore()
	r := resolver.New(s, true, "BR", testLogger())
	evt := model.InboundEvent{
		Phone:       "+5511999990003",
		DisplayName: "Dave",
		Timestamp:   time.Now(),
	}
	result, err := r.Resolve(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != "created" {
		t.Errorf("expected match_type=created, got %q", result.MatchType)
	}
	if result.Confidence != 0.5 {
		t.Errorf("expected confidence=0.5, got %f", result.Confidence)
	}
	if result.Contact == nil {
		t.Fatal("expected contact to be non-nil")
	}
	if result.Contact.Status != "pending" {
		t.Errorf("expected status=pending, got %q", result.Contact.Status)
	}
}

func TestResolve_NotFound(t *testing.T) {
	s := newMockStore()
	r := resolver.New(s, false, "BR", testLogger())
	evt := model.InboundEvent{
		Phone:     "+5511999990099",
		Timestamp: time.Now(),
	}
	result, err := r.Resolve(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != "not_found" {
		t.Errorf("expected match_type=not_found, got %q", result.MatchType)
	}
	if result.Contact != nil {
		t.Errorf("expected nil contact")
	}
}

func TestLookup_ByPhone(t *testing.T) {
	s := newMockStore()
	c := model.NewContact("+5511999990010", "Eve")
	s.add(c)

	r := resolver.New(s, false, "BR", testLogger())
	result, err := r.Lookup(context.Background(), "+55 11 99999-0010")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != "phone" {
		t.Errorf("expected match_type=phone, got %q", result.MatchType)
	}
}

func TestLookup_ByBSUID(t *testing.T) {
	s := newMockStore()
	bsuID := "BR.frank12345678" // valid BSUID
	c := model.NewContact("+5511999990011", "Frank")
	c.BSUID = strPtr(bsuID)
	s.add(c)

	r := resolver.New(s, false, "BR", testLogger())
	result, err := r.Lookup(context.Background(), bsuID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != "bsuid" {
		t.Errorf("expected match_type=bsuid, got %q", result.MatchType)
	}
	if result.Contact == nil || result.Contact.ID != c.ID {
		t.Errorf("expected contact %s", c.ID)
	}
}

func TestLookup_NotFound(t *testing.T) {
	s := newMockStore()
	r := resolver.New(s, false, "BR", testLogger())
	result, err := r.Lookup(context.Background(), "+5511000000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != "not_found" {
		t.Errorf("expected not_found, got %q", result.MatchType)
	}
}

func TestResolve_AutoCreate_WithDisplayName(t *testing.T) {
	s := newMockStore()
	r := resolver.New(s, true, "BR", testLogger())
	waID := "5511999990050@c.us"
	evt := model.InboundEvent{
		Phone:       "+5511999990050",
		DisplayName: "Grace",
		WhatsAppID:  strPtr(waID),
		Timestamp:   time.Now(),
	}
	result, err := r.Resolve(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != "created" {
		t.Errorf("expected match_type=created, got %q", result.MatchType)
	}
	if result.Contact.Name != "Grace" {
		t.Errorf("expected Name=Grace, got %q", result.Contact.Name)
	}
	if result.Contact.WhatsAppID == nil || *result.Contact.WhatsAppID != waID {
		t.Errorf("expected WhatsAppID=%s, got %v", waID, result.Contact.WhatsAppID)
	}
	// Metadata should have whatsapp_name set.
	if len(result.Contact.Metadata) == 0 {
		t.Error("expected metadata to be non-empty")
	}
}

func TestResolve_ExistingContact_MetadataWhatsAppName(t *testing.T) {
	s := newMockStore()
	c := model.NewContact("+5511999990051", "")
	s.add(c)

	r := resolver.New(s, false, "BR", testLogger())
	evt := model.InboundEvent{
		Phone:       "+5511999990051",
		DisplayName: "Henry",
		Timestamp:   time.Now(),
	}
	result, err := r.Resolve(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != "phone" {
		t.Errorf("expected match_type=phone, got %q", result.MatchType)
	}
	// Name should be backfilled.
	if result.Contact.Name != "Henry" {
		t.Errorf("expected Name=Henry, got %q", result.Contact.Name)
	}
	// Metadata should have whatsapp_name.
	if len(result.Contact.Metadata) == 0 {
		t.Error("expected metadata to be non-empty")
	}
}

func TestResolve_ExistingContact_WhatsAppIDBackfill(t *testing.T) {
	s := newMockStore()
	c := model.NewContact("+5511999990052", "Ivan")
	s.add(c)

	waID := "5511999990052@s.whatsapp.net"
	r := resolver.New(s, false, "BR", testLogger())
	evt := model.InboundEvent{
		Phone:      "+5511999990052",
		WhatsAppID: strPtr(waID),
		Timestamp:  time.Now(),
	}
	result, err := r.Resolve(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Contact.WhatsAppID == nil || *result.Contact.WhatsAppID != waID {
		t.Errorf("expected WhatsAppID=%s to be backfilled, got %v", waID, result.Contact.WhatsAppID)
	}
}
