# Dual-Key Identity Resolution

## Status

Accepted

## Context and Problem Statement

WhatsApp has historically addressed contacts by phone number (E.164 format). Meta is rolling out Business Suite User IDs (BSUIDs) — opaque, stable identifiers that replace phone numbers as the primary addressing scheme — broadly by June 2026.

How should WAID handle identity resolution during and after this transition, given that messages from the same human will arrive with a phone number today and a BSUID tomorrow?

## Decision Drivers

* Forward-compatibility with Meta BSUID rollout (June 2026)
* Zero duplicate contacts during the transition period
* Graceful degradation for adapters that don't yet surface BSUIDs
* Minimal operational overhead (no separate deduplication service)

## Considered Options

1. Dual-key resolution (phone + BSUID) from day 1
2. Phone-only, migrate later
3. BSUID-only
4. External deduplication service

## Decision Outcome

Chosen option: **"Dual-key resolution from day 1"**, because it is forward-compatible without schema migrations, prevents duplicate contacts during the transition, and keeps the system self-contained.

### Consequences

#### Good

* No schema migration required when Meta completes the BSUID rollout
* Zero duplicate contacts during the transition — the enrichment path merges phone-keyed and BSUID-keyed records automatically
* Callers always receive a stable `Contact.id` (UUID) regardless of which key was used, decoupling downstream systems from WhatsApp addressing changes

#### Bad

* The resolver has two index lookups per event instead of one; acceptable at current scale
* `bsuid` being nullable requires explicit nil-checks throughout the codebase
* Adapter authors must implement BSUID extraction even for gateways that don't yet surface it (return nil for now)

## Implementation Details

### Schema

The `contacts` table carries both a `phone` field (E.164, always required) and a `bsuid` field (nullable). Both fields are uniquely indexed.

### Resolver Order

1. BSUID match (if present in the event)
2. Phone match (fallback)
3. Auto-create (if `resolver.auto_create = true` and no match found)

The `IdentityResult.match_type` field records which key matched: `bsuid`, `phone`, `created`, or `not_found`.

### Adapter Normalization

Each adapter extracts both phone and BSUID from the raw webhook payload. Adapters that cannot supply a BSUID leave it nil; the resolver degrades gracefully to phone-only matching.

### Enrichment Path

When a phone-matched contact later arrives with a BSUID, the resolver writes the BSUID back to the contact record, linking the two keys permanently.

## Rejected Options

### Phone-only, migrate later

Simpler initially, but requires a painful data migration and possible downtime window when BSUIDs become mandatory.

### BSUID-only

Clean long-term design, but non-functional until Meta completes the rollout.

### External deduplication service

Offload merging to a separate service. Adds operational complexity without clear benefit at this stage.
