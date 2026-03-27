package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/prenansantana/waid/internal/adapter"
	"github.com/prenansantana/waid/internal/model"
)

var adapterRegistry = adapter.DefaultRegistry()

// inboundHandler handles POST /inbound/{source} webhook events.
func (s *Server) inboundHandler(w http.ResponseWriter, r *http.Request) {
	source := chi.URLParam(r, "source")

	a, ok := adapterRegistry.Get(source)
	if !ok {
		respondError(w, http.StatusNotFound, "unknown source: "+source)
		return
	}

	event, err := a.ParseWebhook(r)
	if err != nil {
		s.logger.Error("failed to parse webhook", "source", source, "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.resolver.Resolve(r.Context(), *event)
	if err != nil {
		s.logger.Error("failed to resolve identity", "source", source, "error", err)
		respondError(w, http.StatusInternalServerError, "identity resolution failed")
		return
	}

	var eventType string
	switch result.MatchType {
	case "phone", "bsuid":
		eventType = model.EventContactResolved
	case "created":
		eventType = model.EventContactCreated
	default:
		eventType = model.EventContactNotFound
	}

	identityEvent := model.IdentityEvent{
		Type:      eventType,
		Contact:   result.Contact,
		Phone:     event.Phone,
		Source:    source,
		Timestamp: time.Now().UTC(),
	}

	if s.natsClient != nil {
		subject := fmt.Sprintf("waid.identity.%s", eventType)
		payload, merr := json.Marshal(identityEvent)
		if merr == nil {
			if perr := s.natsClient.Publish(subject, payload); perr != nil {
				s.logger.Warn("failed to publish to NATS", "subject", subject, "error", perr)
			}
		}
	}

	if s.notifier != nil {
		go func() {
			if nerr := s.notifier.Notify(r.Context(), identityEvent); nerr != nil {
				s.logger.Warn("notifier error", "error", nerr)
			}
		}()
	}

	respondJSON(w, http.StatusOK, result)
}

// metaVerifyHandler handles GET /inbound/meta for Meta webhook verification.
func (s *Server) metaVerifyHandler(w http.ResponseWriter, r *http.Request) {
	metaAdapter := &adapter.MetaAdapter{}
	challenge, err := metaAdapter.VerifyWebhook(r, s.config.Meta.VerifyToken)
	if err != nil {
		respondError(w, http.StatusForbidden, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(challenge))
}
