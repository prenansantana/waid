// Package resolver implements the identity resolution engine.
package resolver

import (
	"context"
	"encoding/json"
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
	store          store.Store
	autoCreate     bool
	DefaultCountry string
	logger         *slog.Logger
}

// New constructs a Resolver with the given store, auto-create flag, default country, and logger.
func New(s store.Store, autoCreate bool, defaultCountry string, logger *slog.Logger) *Resolver {
	return &Resolver{
		store:          s,
		autoCreate:     autoCreate,
		DefaultCountry: defaultCountry,
		logger:         logger,
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
	normalized, err := phone.Normalize(evt.Phone, r.DefaultCountry)
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
		changed := false

		// Backfill BSUID if the event carries one but the contact doesn't.
		if evt.BSUID != nil && *evt.BSUID != "" && (contact.BSUID == nil || *contact.BSUID == "") {
			contact.BSUID = evt.BSUID
			changed = true
		}

		// Backfill Name if contact has none but event has a display name.
		if contact.Name == "" && evt.DisplayName != "" {
			contact.Name = evt.DisplayName
			changed = true
		}

		// Backfill WhatsAppID if contact has none but event has one.
		if contact.WhatsAppID == nil && evt.WhatsAppID != nil {
			contact.WhatsAppID = evt.WhatsAppID
			changed = true
		}

		// Update metadata.whatsapp_name if DisplayName is set.
		if evt.DisplayName != "" {
			meta, metaChanged := setMetadataKey(contact.Metadata, "whatsapp_name", evt.DisplayName)
			if metaChanged {
				contact.Metadata = meta
				changed = true
			}
		}

		if changed {
			contact.UpdatedAt = now
			if updateErr := r.store.Update(ctx, contact); updateErr != nil {
				r.logger.Warn("failed to update contact", slog.String("contact_id", contact.ID), slog.String("error", updateErr.Error()))
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
		if evt.WhatsAppID != nil {
			newContact.WhatsAppID = evt.WhatsAppID
		}
		if evt.DisplayName != "" {
			meta, _ := setMetadataKey(nil, "whatsapp_name", evt.DisplayName)
			newContact.Metadata = meta
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
		normalized, err := phone.Normalize(phoneOrID, r.DefaultCountry)
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

// setMetadataKey unmarshals the given JSON metadata, sets key=value, marshals back.
// Returns the new JSON and whether the value actually changed.
func setMetadataKey(raw json.RawMessage, key, value string) (json.RawMessage, bool) {
	m := make(map[string]interface{})
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &m)
	}
	existing, ok := m[key]
	if ok && existing == value {
		return raw, false
	}
	m[key] = value
	out, err := json.Marshal(m)
	if err != nil {
		return raw, false
	}
	return json.RawMessage(out), true
}
