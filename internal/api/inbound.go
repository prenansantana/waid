package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/prenansantana/waid/internal/adapter"
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

	respondJSON(w, http.StatusOK, event)
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
