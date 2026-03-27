package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/prenansantana/waid/internal/model"
	"github.com/prenansantana/waid/internal/store"
	"github.com/prenansantana/waid/pkg/phone"
)

// healthCheck returns service liveness information.
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if err := s.store.Ping(r.Context()); err != nil {
		dbStatus = "error"
	}
	respondJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"database": dbStatus,
		"version":  "dev",
	})
}

// resolveHandler resolves a phone number or BSUID to an identity result.
func (s *Server) resolveHandler(w http.ResponseWriter, r *http.Request) {
	phoneOrID := chi.URLParam(r, "phone_or_id")
	result, err := s.resolver.Lookup(r.Context(), phoneOrID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// createContactRequest is the JSON body for POST /contacts.
type createContactRequest struct {
	Phone      string          `json:"phone"`
	Name       string          `json:"name"`
	ExternalID *string         `json:"external_id,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// createContact creates a new contact.
func (s *Server) createContact(w http.ResponseWriter, r *http.Request) {
	var req createContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	normalized, err := phone.Normalize(req.Phone)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid phone number")
		return
	}

	c := model.NewContact(normalized, req.Name)
	c.ExternalID = req.ExternalID
	if len(req.Metadata) > 0 {
		c.Metadata = req.Metadata
	}

	if err := s.store.Create(r.Context(), c); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, c)
}

// paginatedResponse wraps a list result with pagination metadata.
type paginatedResponse struct {
	Data    interface{} `json:"data"`
	Total   int         `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
}

// listContacts returns a paginated list of contacts.
func (s *Server) listContacts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 50
	}

	opts := store.ListOpts{
		Page:    page,
		PerPage: perPage,
		Query:   q.Get("q"),
	}

	contacts, total, err := s.store.List(r.Context(), opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, paginatedResponse{
		Data:    contacts,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

// getContact returns a single contact by ID.
func (s *Server) getContact(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := s.store.FindByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "contact not found")
		return
	}
	respondJSON(w, http.StatusOK, c)
}

// updateContactRequest is the JSON body for PUT /contacts/{id}.
type updateContactRequest struct {
	Name       *string         `json:"name,omitempty"`
	ExternalID *string         `json:"external_id,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	Status     *string         `json:"status,omitempty"`
}

// updateContact merges provided fields into an existing contact.
func (s *Server) updateContact(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := s.store.FindByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "contact not found")
		return
	}

	var req updateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.ExternalID != nil {
		c.ExternalID = req.ExternalID
	}
	if len(req.Metadata) > 0 {
		c.Metadata = req.Metadata
	}
	if req.Status != nil {
		c.Status = *req.Status
	}
	c.UpdatedAt = time.Now().UTC()

	if err := s.store.Update(r.Context(), c); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, c)
}

// deleteContact soft-deletes a contact.
func (s *Server) deleteContact(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusNotFound, "contact not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// importContacts parses a multipart form file and bulk-upserts contacts.
func (s *Server) importContacts(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "missing 'file' field in form")
		return
	}
	defer file.Close()

	contacts, err := parseContacts(file, header.Filename, header.Header.Get("Content-Type"))
	if err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	report, err := s.store.BulkUpsert(r.Context(), contacts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, report)
}

// createWebhookRequest is the JSON body for POST /webhooks.
type createWebhookRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"`
}

// createWebhook registers a new webhook target.
func (s *Server) createWebhook(w http.ResponseWriter, r *http.Request) {
	if s.webhookStore == nil {
		respondError(w, http.StatusNotImplemented, "webhook store not configured")
		return
	}
	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "url is required")
		return
	}
	wh := &model.WebhookTarget{
		URL:    req.URL,
		Events: req.Events,
		Secret: req.Secret,
	}
	if err := s.webhookStore.CreateWebhook(r.Context(), wh); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, wh)
}

// listWebhooks returns all active webhook targets.
func (s *Server) listWebhooks(w http.ResponseWriter, r *http.Request) {
	if s.webhookStore == nil {
		respondError(w, http.StatusNotImplemented, "webhook store not configured")
		return
	}
	webhooks, err := s.webhookStore.ListWebhooks(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, webhooks)
}

// deleteWebhook removes a webhook target by ID.
func (s *Server) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	if s.webhookStore == nil {
		respondError(w, http.StatusNotImplemented, "webhook store not configured")
		return
	}
	id := chi.URLParam(r, "id")
	if err := s.webhookStore.DeleteWebhook(r.Context(), id); err != nil {
		respondError(w, http.StatusNotFound, "webhook not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
