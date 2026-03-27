# WAID — WhatsApp Identity Resolver

> Open-source middleware that resolves WhatsApp contacts to your business customer records.

[![Go](https://img.shields.io/badge/go-1.22+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![CI](https://github.com/prenansantana/waid/actions/workflows/ci.yml/badge.svg)](https://github.com/prenansantana/waid/actions/workflows/ci.yml)

---

## The Problem

Your business receives hundreds of WhatsApp messages every day. A message arrives from +55 (62) 98576-4545 — but who is this person in your CRM? Is it a patient, a customer, a lead? Without instant identity resolution, your team wastes time looking them up manually, or worse, responds without context.

**WAID** sits between your WhatsApp gateway and your application. When a message arrives, WAID instantly resolves the sender's phone number to a contact record in your database and notifies your systems via webhook — in under 5ms.

---

## Key Features

- **Identity resolution in <5ms** — phone number maps to your customer record before your handler even starts
- **Multi-source adapters** — works with WAHA, Evolution API, Meta Cloud API, and a generic webhook format
- **Smart phone normalization** — handles local numbers like `(62) 98576-4545` with a configurable default country code
- **Auto-create contacts** — unknown numbers automatically become pending contacts with the WhatsApp display name
- **Real-time webhooks** — fires `contact.resolved`, `contact.created`, `contact.not_found` to your registered endpoints
- **WhatsApp metadata capture** — always stores `whatsapp_name` and `whatsapp_photo` on resolution
- **Zero-config mode** — SQLite + embedded NATS, just run the binary
- **Production mode** — PostgreSQL + external NATS for scale
- **Forward-compatible with Meta BSUID** — when Meta rolls out opaque user IDs, WAID handles them without schema changes

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│          Adapters (inbound webhook layer)        │
│  WAHA · Evolution API · Meta Cloud API · Generic │
└────────────────────┬────────────────────────────┘
                     │ POST /inbound/{source}
                     ▼
┌─────────────────────────────────────────────────┐
│               Resolver Engine                   │
│  1. Normalize phone (local → E.164)             │
│  2. Try BSUID lookup (if present)               │
│  3. Phone lookup against contact registry       │
│  4. Auto-create if resolver.auto_create = true  │
└──────────┬──────────────────────────┬───────────┘
           │                          │
           ▼                          ▼
┌──────────────────┐       ┌──────────────────────┐
│  Store           │       │  Notifier            │
│  SQLite / PG     │       │  Webhooks (HMAC)     │
│                  │       │  NATS events         │
└──────────────────┘       └──────────────────────┘
```

---

## Real-World Example: Medical Clinic

A clinic receives 500+ WhatsApp messages per day from patients scheduling appointments, asking about results, or requesting prescriptions.

**Setup (one time):**

1. Import your patient database — WAID accepts CSV with local phone numbers:
   ```bash
   curl -X POST http://localhost:8080/import \
     -H "X-API-Key: your-key" \
     -F "file=@patients.csv"
   ```

2. Register your CRM webhook so WAID notifies it on every resolution:
   ```bash
   curl -X POST http://localhost:8080/webhooks \
     -H "X-API-Key: your-key" \
     -H "Content-Type: application/json" \
     -d '{"url":"https://crm.clinic.com/hooks/waid","events":["contact.resolved","contact.created"]}'
   ```

3. Point your WhatsApp gateway (e.g. WAHA) at WAID's inbound endpoint.

**Runtime — patient sends a message:**

```
Patient: +55 (62) 98576-4545  →  WAID resolver
WAID normalizes: +5562985764545
WAID matches: Maria Oliveira, patient_id=8821, last_visit=2026-02-10
WAID fires webhook → CRM opens patient record in <5ms
```

**Unknown number:**

```
Unregistered caller  →  WAID auto-creates pending contact
whatsapp_name: "João"  →  CRM receives contact.created event
Staff follows up and links to full record
```

---

## Quickstart

### Docker

```bash
docker run -p 8080:8080 ghcr.io/prenansantana/waid
```

### Binary

Download the latest release from [GitHub Releases](https://github.com/prenansantana/waid/releases), then:

```bash
./waid
```

### Docker Compose

```bash
docker compose -f deploy/docker/docker-compose.yml up
```

---

## Configuration

Configuration is read from `waid.yaml` (searched in `.`, `~/.waid`, `/etc/waid`) and overridden by environment variables. Environment variables take full precedence.

Copy the example file to get started:

```bash
cp waid.yaml.example waid.yaml
```

### Reference

| Environment Variable                | YAML key                    | Default                    | Description                                                                      |
|-------------------------------------|-----------------------------|----------------------------|----------------------------------------------------------------------------------|
| `WAID_SERVER_PORT`                  | `server.port`               | `8080`                     | HTTP listen port                                                                 |
| `WAID_SERVER_API_KEY`               | `server.api_key`            | *(empty — auth disabled)*  | API key sent via `X-API-Key` header                                              |
| `WAID_SERVER_CORS_ORIGINS`          | `server.cors_origins`       | `["*"]`                    | Allowed CORS origins (e.g. `["https://app.example.com"]`)                        |
| `WAID_DATABASE_DRIVER`              | `database.driver`           | `sqlite`                   | Database backend: `sqlite` or `postgres`                                         |
| `WAID_DATABASE_URL`                 | `database.url`              | `waid.db`                  | File path (SQLite) or connection string (PostgreSQL)                             |
| `WAID_NATS_EMBEDDED`                | `nats.embedded`             | `true`                     | Run an embedded NATS server instead of connecting to one                         |
| `WAID_NATS_URL`                     | `nats.url`                  | `nats://localhost:4222`    | NATS server URL (used when `embedded` is `false`)                                |
| `WAID_RESOLVER_AUTO_CREATE`         | `resolver.auto_create`      | `true`                     | Auto-create a contact when an unknown identity is seen                           |
| `WAID_RESOLVER_DEFAULT_COUNTRY`     | `resolver.default_country`  | `BR`                       | ISO 3166-1 alpha-2 country code used when normalizing local phone numbers        |
| `WAID_LOGGING_LEVEL`                | `logging.level`             | `info`                     | Log level: `debug`, `info`, `warn`, `error`                                      |
| `WAID_LOGGING_FORMAT`               | `logging.format`            | `json`                     | Log format: `json` or `text`                                                     |
| `WAID_META_VERIFY_TOKEN`            | `meta.verify_token`         | *(empty)*                  | Token for Meta Cloud API webhook verification handshake                          |

### Example `waid.yaml`

```yaml
server:
  port: 8080
  api_key: "change-me"
  cors_origins:
    - "https://app.example.com"

database:
  driver: sqlite
  url: waid.db
  # driver: postgres
  # url: "postgres://user:pass@localhost:5432/waid?sslmode=disable"

nats:
  embedded: true
  url: "nats://localhost:4222"

resolver:
  auto_create: true
  default_country: "BR"

logging:
  level: info
  format: json
```

---

## API Reference

All endpoints except `/inbound/*` require the `X-API-Key` header when `WAID_SERVER_API_KEY` is set.

### Health

#### `GET /health`

Returns service liveness and database connectivity.

**Response `200 OK`**
```json
{
  "status": "ok",
  "database": "ok",
  "version": "dev"
}
```

---

### Identity Resolution

#### `GET /resolve/{phone_or_id}`

Resolves a phone number or BSUID to a contact identity. The resolver normalizes the phone to E.164 using the configured default country, tries BSUID first (if provided), then phone. If `resolver.auto_create` is enabled and no match is found, a new contact is created.

| Parameter     | Type   | Description                          |
|---------------|--------|--------------------------------------|
| `phone_or_id` | string | E.164 phone number or BSUID string   |

**Response `200 OK`**
```json
{
  "contact": {
    "id": "018e1234-...",
    "phone": "+5511999990000",
    "bsuid": "ABCD1234",
    "name": "Alice",
    "status": "active",
    "created_at": "2026-03-01T12:00:00Z",
    "updated_at": "2026-03-01T12:00:00Z"
  },
  "match_type": "phone",
  "confidence": 1.0,
  "resolved_at": "2026-03-27T10:00:00Z"
}
```

`match_type` values: `phone`, `bsuid`, `created`, `not_found`

---

### Contacts

#### `POST /contacts`

Creates a new contact.

**Request body**
```json
{
  "phone": "+5511999990000",
  "name": "Alice",
  "external_id": "crm-42",
  "metadata": { "tag": "vip" }
}
```

**Response `201 Created`** — returns the created `Contact` object.

---

#### `GET /contacts`

Returns a paginated list of contacts.

| Query param | Default | Description                    |
|-------------|---------|--------------------------------|
| `page`      | `1`     | Page number                    |
| `per_page`  | `50`    | Results per page               |
| `q`         | —       | Search query (name/phone)      |

**Response `200 OK`**
```json
{
  "data": [ /* Contact objects */ ],
  "total": 120,
  "page": 1,
  "per_page": 50
}
```

---

#### `GET /contacts/{id}`

Returns a single contact by UUID.

**Response `200 OK`** — returns the `Contact` object.
**Response `404 Not Found`** — contact not found.

---

#### `PUT /contacts/{id}`

Partial update of a contact. Only provided fields are changed.

**Request body** (all fields optional)
```json
{
  "name": "Alice Smith",
  "external_id": "crm-42",
  "status": "active",
  "metadata": { "tag": "vip" }
}
```

**Response `200 OK`** — returns the updated `Contact` object.

---

#### `DELETE /contacts/{id}`

Soft-deletes a contact.

**Response `204 No Content`**

---

### Import

#### `POST /import`

Bulk-upserts contacts from a CSV or JSON file upload. Phone numbers are normalized using `WAID_RESOLVER_DEFAULT_COUNTRY` for local numbers without a country code.

**Request** — `multipart/form-data` with a `file` field containing a CSV or JSON file.

**Response `200 OK`**
```json
{
  "total": 500,
  "created": 480,
  "updated": 15,
  "errors": 5,
  "details": [
    { "row": 12, "phone": "invalid", "reason": "invalid phone number" }
  ]
}
```

---

### Inbound Webhooks

#### `POST /inbound/{source}`

Receives a raw webhook event from a WhatsApp gateway adapter, resolves the sender identity, and fires outbound webhooks to registered targets. No API key required.

| Parameter | Description                                        |
|-----------|----------------------------------------------------|
| `source`  | Adapter name: `waha`, `evolution`, `meta`, `generic` |

The body format depends on the adapter. WAID normalizes it to an `InboundEvent` internally.

**Response `200 OK`**
```json
{
  "source_id": "wamid.xxx",
  "phone": "+5511999990000",
  "bsuid": "ABCD1234",
  "display_name": "Alice",
  "source": "waha",
  "timestamp": "2026-03-27T10:00:00Z"
}
```

---

#### `GET /inbound/meta`

Meta Cloud API webhook verification handshake. Meta sends a `GET` request with `hub.challenge` — WAID verifies `hub.verify_token` against `WAID_META_VERIFY_TOKEN` and echoes the challenge.

---

### Webhooks

#### `POST /webhooks`

Registers a new webhook target. WAID will POST to this URL on every identity resolution event, signed with HMAC-SHA256 using the provided secret.

**Request body**
```json
{
  "url": "https://your-service.example/hook",
  "events": ["contact.resolved", "contact.created"],
  "secret": "hmac-signing-secret"
}
```

Event types: `contact.resolved`, `contact.created`, `contact.updated`, `contact.not_found`

**Response `201 Created`** — returns the created `WebhookTarget` object.

---

#### `GET /webhooks`

Lists all active webhook targets.

**Response `200 OK`** — returns an array of `WebhookTarget` objects.

---

#### `DELETE /webhooks/{id}`

Removes a webhook target by ID.

**Response `204 No Content`**

---

## SDKs

| Language   | Package                                                        |
|------------|----------------------------------------------------------------|
| TypeScript | `npm install @waid/client` *(coming soon)*                    |
| Python     | `pip install waid-client` *(coming soon)*                     |
| Go         | `go get github.com/prenansantana/waid/sdk/go` *(coming soon)* |

---

## Tech Stack

| Component       | Technology                                              |
|-----------------|---------------------------------------------------------|
| Language        | Go 1.22+                                                |
| HTTP router     | [chi](https://github.com/go-chi/chi)                    |
| Database        | SQLite (default) / PostgreSQL                           |
| Messaging       | NATS (embedded or external)                             |
| Configuration   | [Viper](https://github.com/spf13/viper)                 |
| Phone parsing   | Internal `pkg/phone` (E.164 normalization)              |
| Container image | `ghcr.io/prenansantana/waid`                            |

---

## License

MIT — see [LICENSE](LICENSE).
