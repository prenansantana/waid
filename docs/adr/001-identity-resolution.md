# WhatsApp Identity Resolution Strategy

## Status

Accepted

## Context and Problem Statement

Businesses using WhatsApp receive messages from contacts whose phone numbers often arrive in local formats (e.g. `(62) 98576-4545` without a country code), with a JID suffix (`5511999990000@s.whatsapp.net`), or through different gateway providers that each have their own payload schema. At the same time, the same contact may reach the business through WAHA today and Meta Cloud API tomorrow.

How should WAID resolve an inbound WhatsApp event to a stable, canonical contact record in the business's customer database, given these normalization challenges and the diversity of source adapters?

## Decision Drivers

- Phone normalization complexity — local numbers, national prefix variations (Brazilian 8→9 digit), JID suffixes, and missing country codes all require smart normalization before matching
- Multi-source adapter abstraction — WAHA, Evolution API, Meta Cloud API, and generic webhooks produce different payload shapes; the resolver must be adapter-agnostic
- Real-time resolution performance — the identity lookup must complete in under 5ms to be useful in a live messaging flow
- Auto-create behavior — unknown contacts should not silently fail; the service must capture the WhatsApp display name and create a pending contact record
- Forward compatibility with Meta BSUID — Meta plans to roll out opaque Business Suite User IDs (BSUIDs) that will eventually replace phone-number addressing; the schema must accommodate this without a future migration

## Considered Options

1. **Normalize-and-match** — normalize every inbound phone number to E.164 using a configurable default country, then match against the contact registry by phone; store BSUID as an optional secondary key for future enrichment
2. **JID-based matching** — store raw WhatsApp JIDs (`number@s.whatsapp.net`) as the primary key and match on those
3. **External identity service** — delegate deduplication and resolution to a third-party identity service

## Decision Outcome

Chosen option: **Normalize-and-match (Option 1) with BSUID forward-compatibility**, because it solves the real problem today (phone number normalization across formats and gateways) while keeping the path to BSUID adoption open without a schema migration.

### Consequences

#### Good

- Phone normalization handles the actual messy data businesses have: local numbers, Brazilian digit expansion, JID suffixes — all resolved to a clean E.164 key
- A configurable `resolver.default_country` (default: `BR`) means operators with a regional customer base do not need to re-format their contact database before import
- Callers always receive a stable `Contact.id` (UUID) regardless of which key was used, decoupling downstream CRM systems from WhatsApp addressing details
- BSUID is stored as a nullable secondary index; when Meta completes the rollout, the resolver enriches existing contacts with their BSUID transparently — no schema migration needed
- Auto-create with `whatsapp_name` capture ensures no inbound event is lost, even from unknown numbers

#### Bad

- Two index lookups per event (BSUID check + phone check) instead of one; acceptable at current scale
- `bsuid` being nullable requires explicit nil-checks in the codebase
- Adapter authors must implement BSUID extraction even for gateways that do not yet surface it (return nil for now)

## Implementation Notes

### Phone Normalization

Each inbound event goes through `pkg/phone` normalization before any lookup:

1. Strip JID suffix (`@s.whatsapp.net`, `@g.us`)
2. Remove formatting characters (spaces, dashes, parentheses)
3. If no country code prefix is detected, prepend the configured `default_country` dial code
4. Apply Brazilian 8→9 digit expansion where applicable
5. Validate and output E.164

### Resolver Order

1. If BSUID is present in the event, try BSUID lookup
2. Phone lookup (always, using normalized E.164)
3. If no match and `resolver.auto_create = true`, create a pending contact with `whatsapp_name` and `whatsapp_photo`

The `IdentityResult.match_type` field records which key matched: `bsuid`, `phone`, `created`, or `not_found`.

### BSUID Enrichment Path

When a phone-matched contact later arrives with a BSUID (e.g. after Meta completes the rollout for that user), the resolver writes the BSUID back to the contact record, permanently linking the two keys.

### Adapter Abstraction

Each adapter (`waha`, `evolution`, `meta`, `generic`) implements a common interface that extracts a normalized `InboundEvent` from the raw gateway payload. Adapters that cannot supply a BSUID leave it nil; the resolver degrades gracefully to phone-only matching.

## Rejected Options

### JID-based matching

Storing raw JIDs requires every import and lookup to use JID format, which is not how business databases store phone numbers. It also ties the data model to WhatsApp's internal addressing, making the system harder to use with non-WhatsApp channels in the future.

### External identity service

Offloading resolution to a third party adds a network hop (breaking the <5ms target), introduces an external dependency, and provides no clear benefit for the normalization problem, which is domain-specific to each business's data.
