// Package adapter provides inbound webhook adapters for WhatsApp providers.
package adapter

import (
	"fmt"
	"net/http"

	"github.com/prenansantana/waid/internal/model"
)

// Adapter parses a raw HTTP webhook request into a normalized InboundEvent.
type Adapter interface {
	Name() string
	ParseWebhook(r *http.Request) (*model.InboundEvent, error)
}

// Registry holds named adapters for lookup by source name.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{adapters: make(map[string]Adapter)}
}

// Register adds an adapter to the registry, keyed by its Name().
func (r *Registry) Register(a Adapter) {
	r.adapters[a.Name()] = a
}

// Get returns the adapter with the given name, or false if not found.
func (r *Registry) Get(name string) (Adapter, bool) {
	a, ok := r.adapters[name]
	if !ok {
		return nil, false
	}
	return a, true
}

// DefaultRegistry returns a Registry pre-populated with all built-in adapters.
func DefaultRegistry() *Registry {
	reg := NewRegistry()
	reg.Register(&WAHAAdapter{})
	reg.Register(&EvolutionAdapter{})
	reg.Register(&MetaAdapter{})
	reg.Register(&GenericAdapter{})
	return reg
}

// errMissingField returns a consistent error for missing required payload fields.
func errMissingField(field string) error {
	return fmt.Errorf("adapter: missing required field %q", field)
}
