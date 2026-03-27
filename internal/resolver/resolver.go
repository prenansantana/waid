// Package resolver implements the identity resolution engine.
package resolver

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/prenansantana/waid/internal/model"
	"github.com/prenansantana/waid/internal/store"
	pkgbsuid "github.com/prenansantana/waid/pkg/bsuid"
	"github.com/prenansantana/waid/pkg/phone"
)

// ErrNotFound is returned when no contact matches the lookup criteria.
var ErrNotFound = errors.New("resolver: contact not found")

// Resolver resolves inbound events to known Contact records.
type Resolver struct {
	store      store.Store
	autoCreate bool
	logger     *slog.Logger
}

// New constructs a Resolver with the given store, auto-create flag, and logger.
func New(s store.Store, autoCreate bool, logger *slog.Logger) *Resolver {
	return &Resolver{
		store:      s,
		autoCreate: autoCreate,
		logger:     logger,
	}
}

// Resolve attempts to match an InboundEvent to an existing Contact.
// Resolution order:
//  1. BSUID lookup (if provided)
//  2. Phone lookup (normalized)
//  3. Auto-create (if enabled)
//  4. not_found
func (r *Resolver) Resolve(ctx context.Context, evt model.InboundEvent) (*model.IdentityResult, error) {
	now := time.Now().UTC()

	// 1. Normalize phone number.
	normalized, err := phone.Normalize(evt.Phone)
	if err != nil {
		return nil, err
	}

	// 2. BSUID lookup.
	if evt.BSUID != nil && *evt.BSUID != "" {
		contact, err := r.store.FindByBSUID(ctx, *evt.BSUID)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return nil, err
		}
		if err == nil && contact != nil {
			return &model.IdentityResult{
				Contact:    contact,
				MatchType:  "bsuid",
				Confidence: 1.0,
				ResolvedAt: now,
			}, nil
		}
	}

	// 3. Phone lookup.
	contact, err := r.store.FindByPhone(ctx, normalized)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	if err == nil && contact != nil {
		// Backfill BSUID if the event carries one but the contact doesn't.
		if evt.BSUID != nil && *evt.BSUID != "" && (contact.BSUID == nil || *contact.BSUID == "") {
			contact.BSUID = evt.BSUID
			contact.UpdatedAt = now
			if updateErr := r.store.Update(ctx, contact); updateErr != nil {
				r.logger.Warn("failed to backfill bsuid", slog.String("contact_id", contact.ID), slog.String("error", updateErr.Error()))
			}
		}
		return &model.IdentityResult{
			Contact:    contact,
			MatchType:  "phone",
			Confidence: 1.0,
			ResolvedAt: now,
		}, nil
	}

	// 4. Auto-create.
	if r.autoCreate {
		newContact := model.NewContact(normalized, evt.DisplayName)
		newContact.Status = "pending"
		if evt.BSUID != nil && *evt.BSUID != "" {
			newContact.BSUID = evt.BSUID
		}
		if createErr := r.store.Create(ctx, newContact); createErr != nil {
			return nil, createErr
		}
		r.logger.Info("auto-created contact", slog.String("contact_id", newContact.ID), slog.String("phone", normalized))
		return &model.IdentityResult{
			Contact:    newContact,
			MatchType:  "created",
			Confidence: 0.5,
			ResolvedAt: now,
		}, nil
	}

	// 5. Not found.
	return &model.IdentityResult{
		Contact:    nil,
		MatchType:  "not_found",
		Confidence: 0.0,
		ResolvedAt: now,
	}, nil
}

// Lookup resolves a phone number or BSUID to a Contact.
// If the input looks like a BSUID it uses FindByBSUID; otherwise it normalizes
// as a phone number and uses FindByPhone.
func (r *Resolver) Lookup(ctx context.Context, phoneOrID string) (*model.IdentityResult, error) {
	now := time.Now().UTC()

	if pkgbsuid.IsBSUID(phoneOrID) {
		contact, err := r.store.FindByBSUID(ctx, phoneOrID)
		if err == nil && contact != nil {
			return &model.IdentityResult{
				Contact:    contact,
				MatchType:  "bsuid",
				Confidence: 1.0,
				ResolvedAt: now,
			}, nil
		}
	} else {
		normalized, err := phone.Normalize(phoneOrID)
		if err == nil {
			contact, err := r.store.FindByPhone(ctx, normalized)
			if err == nil && contact != nil {
				return &model.IdentityResult{
					Contact:    contact,
					MatchType:  "phone",
					Confidence: 1.0,
					ResolvedAt: now,
				}, nil
			}
		}
	}

	return &model.IdentityResult{
		Contact:    nil,
		MatchType:  "not_found",
		Confidence: 0.0,
		ResolvedAt: now,
	}, nil
}
