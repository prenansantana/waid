// Package api provides HTTP handlers and route registration.
package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/prenansantana/waid/internal/config"
	"github.com/prenansantana/waid/internal/notifier"
	"github.com/prenansantana/waid/internal/resolver"
	"github.com/prenansantana/waid/internal/store"
)

// Server holds the HTTP router and its dependencies.
type Server struct {
	router       chi.Router
	resolver     *resolver.Resolver
	store        store.Store
	webhookStore notifier.WebhookStore
	config       *config.Config
	logger       *slog.Logger
}

// New constructs a Server and registers all routes.
func New(cfg *config.Config, s store.Store, r *resolver.Resolver, logger *slog.Logger) *Server {
	return NewWithWebhookStore(cfg, s, r, nil, logger)
}

// NewWithWebhookStore constructs a Server with a webhook store for CRUD operations.
func NewWithWebhookStore(cfg *config.Config, s store.Store, r *resolver.Resolver, ws notifier.WebhookStore, logger *slog.Logger) *Server {
	srv := &Server{
		router:       chi.NewRouter(),
		resolver:     r,
		store:        s,
		webhookStore: ws,
		config:       cfg,
		logger:       logger,
	}
	srv.routes()
	return srv
}

// Handler returns the underlying http.Handler (the chi router).
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) routes() {
	r := s.router

	// Global middleware.
	r.Use(Recovery)
	r.Use(RequestLogger(s.logger))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(APIKeyAuth(s.config.Server.APIKey))

	// Routes.
	r.Get("/health", s.healthCheck)
	r.Get("/resolve/{phone_or_id}", s.resolveHandler)

	r.Route("/contacts", func(r chi.Router) {
		r.Post("/", s.createContact)
		r.Get("/", s.listContacts)
		r.Get("/{id}", s.getContact)
		r.Put("/{id}", s.updateContact)
		r.Delete("/{id}", s.deleteContact)
	})

	r.Post("/import", s.importContacts)

	// Webhook CRUD.
	r.Post("/webhooks", s.createWebhook)
	r.Get("/webhooks", s.listWebhooks)
	r.Delete("/webhooks/{id}", s.deleteWebhook)

	// Inbound webhook routes.
	r.Post("/inbound/{source}", s.inboundHandler)
	r.Get("/inbound/meta", s.metaVerifyHandler)
}
