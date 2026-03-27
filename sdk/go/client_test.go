package waid_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	waid "github.com/prenansantana/waid-sdk-go"
)

func TestResolve(t *testing.T) {
	want := waid.IdentityResult{
		MatchType:  "phone",
		Confidence: 1.0,
		ResolvedAt: time.Now().UTC(),
		Contact: &waid.Contact{
			ID:    "abc-123",
			Phone: "+5511999990000",
			Name:  "Alice",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/resolve/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("missing or wrong API key")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := waid.NewClient(srv.URL, waid.WithAPIKey("test-key"))
	got, err := client.Resolve(context.Background(), "+5511999990000")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.MatchType != want.MatchType {
		t.Errorf("MatchType: got %q, want %q", got.MatchType, want.MatchType)
	}
	if got.Contact == nil || got.Contact.Phone != "+5511999990000" {
		t.Errorf("unexpected contact: %+v", got.Contact)
	}
}

func TestCreateContact(t *testing.T) {
	want := waid.Contact{
		ID:    "new-uuid",
		Phone: "+5511999990000",
		Name:  "Bob",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/contacts" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body waid.CreateContactInput
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Phone != "+5511999990000" || body.Name != "Bob" {
			t.Errorf("unexpected body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := waid.NewClient(srv.URL)
	got, err := client.CreateContact(context.Background(), waid.CreateContactInput{
		Phone: "+5511999990000",
		Name:  "Bob",
	})
	if err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID: got %q, want %q", got.ID, want.ID)
	}
}

func TestListContacts(t *testing.T) {
	want := waid.PaginatedContacts{
		Data:    []waid.Contact{{ID: "1", Phone: "+1234567890", Name: "Alice"}},
		Total:   1,
		Page:    1,
		PerPage: 50,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/contacts" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("expected page=1, got %q", r.URL.Query().Get("page"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := waid.NewClient(srv.URL)
	got, err := client.ListContacts(context.Background(), waid.ListOpts{Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if got.Total != 1 {
		t.Errorf("Total: got %d, want 1", got.Total)
	}
	if len(got.Data) != 1 || got.Data[0].Name != "Alice" {
		t.Errorf("unexpected data: %+v", got.Data)
	}
}

func TestHealth(t *testing.T) {
	want := waid.HealthStatus{Status: "ok", Database: "ok", Version: "dev"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/health" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := waid.NewClient(srv.URL)
	got, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if got.Status != "ok" || got.Database != "ok" {
		t.Errorf("unexpected health: %+v", got)
	}
}

func TestErrorHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "contact not found"})
	}))
	defer srv.Close()

	client := waid.NewClient(srv.URL)
	_, err := client.GetContact(context.Background(), "does-not-exist")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wErr, ok := err.(*waid.WAIDError)
	if !ok {
		t.Fatalf("expected *WAIDError, got %T: %v", err, err)
	}
	if wErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode: got %d, want 404", wErr.StatusCode)
	}
	if wErr.Message != "contact not found" {
		t.Errorf("Message: got %q, want %q", wErr.Message, "contact not found")
	}
}
