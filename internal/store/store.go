// Package store provides the database layer supporting SQLite and PostgreSQL.
package store

import (
	"context"
	"errors"

	"github.com/prenansantana/waid/internal/model"
)

// ErrNotFound is returned by Find* methods when no matching record exists.
var ErrNotFound = errors.New("not found")

// ListOpts carries pagination and search parameters for list queries.
type ListOpts struct {
	Page    int
	PerPage int
	Query   string
}

// Store is the persistence interface for Contact records.
type Store interface {
	FindByPhone(ctx context.Context, phone string) (*model.Contact, error)
	FindByBSUID(ctx context.Context, bsuid string) (*model.Contact, error)
	FindByExternalID(ctx context.Context, extID string) (*model.Contact, error)
	FindByWhatsAppID(ctx context.Context, waID string) (*model.Contact, error)
	FindByID(ctx context.Context, id string) (*model.Contact, error)
	Create(ctx context.Context, c *model.Contact) error
	Update(ctx context.Context, c *model.Contact) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, opts ListOpts) ([]model.Contact, int, error)
	BulkUpsert(ctx context.Context, contacts []model.Contact) (*model.ImportReport, error)
	Ping(ctx context.Context) error
	Close() error
}
