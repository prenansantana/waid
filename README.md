# WAID — WhatsApp Identity Resolver

> WAHA resolves transport. WAID resolves identity.

[![Go](https://img.shields.io/badge/go-1.22+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![CI](https://github.com/prenansantana/waid/actions/workflows/ci.yml/badge.svg)](https://github.com/prenansantana/waid/actions/workflows/ci.yml)

---

## What is WAID?

WhatsApp is transitioning from phone-number-based addressing to opaque Business Suite User IDs (BSUIDs) — rolling out broadly by June 2026. This creates an identity resolution problem: the same human contact may arrive via phone number today and via BSUID tomorrow.

**WAID** is a lightweight service that sits between your WhatsApp gateway (WAHA, Evolution API, Meta Cloud API, or a generic webhook) and your application. It maintains a contact registry keyed by both phone and BSUID, resolves inbound events to a stable identity, and notifies your systems via webhooks.

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
│  1. Try BSUID lookup                            │
│  2. Fall back to phone lookup                   │
│  3. Auto-create if resolver.auto_create = true  │
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

| Environment Variable      | YAML key                  | Default                    | Description                                               |
|---------------------------|---------------------------|----------------------------|-----------------------------------------------------------|
| `WAID_SERVER_PORT`        | `server.port`             | `8080`                     | HTTP listen port                                          |
| `WAID_SERVER_API_KEY`     | `server.api_key`          | *(empty — auth disabled)*  | API key sent via `X-API-Key` header                       |
| `WAID_DATABASE_DRIVER`    | `database.driver`         | `sqlite`                   | Database backend: `sqlite` or `postgres`                  |
| `WAID_DATABASE_URL`       | `database.url`            | `waid.db`                  | File path (SQLite) or connection string (PostgreSQL)      |
| `WAID_NATS_EMBEDDED`      | `nats.embedded`           | `true`                     | Run an embedded NATS server instead of connecting to one  |
| `WAID_NATS_URL`           | `nats.url`                | `nats://localhost:4222`    | NATS server URL (used when `embedded` is `false`)         |
| `WAID_RESOLVER_AUTO_CREATE` | `resolver.auto_create`  | `true`                     | Auto-create a contact when an unknown identity is seen    |
| `WAID_LOGGING_LEVEL`      | `logging.level`           | `info`                     | Log level: `debug`, `info`, `warn`, `error`               |
| `WAID_LOGGING_FORMAT`     | `logging.format`          | `json`                     | Log format: `json` or `text`                              |
| `WAID_META_VERIFY_TOKEN`  | `meta.verify_token`       | *(empty)*                  | Token for Meta Cloud API webhook verification handshake   |

### Example `waid.yaml`

```yaml
server:
  port: 8080
  api_key: "change-me"

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

logging:
  level: info
  format: json
```

---

## API Reference

All endpoints require the `X-API-Key` header when `WAID_SERVER_API_KEY` is set.

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

Resolves a phone number or BSUID to a contact identity. The resolver tries BSUID first, then phone. If `resolver.auto_create` is enabled and no match is found, a new contact is created.

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

Bulk-upserts contacts from a CSV or JSON file upload.

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

Receives a raw webhook event from a WhatsApp gateway adapter and resolves the identity.

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

Registers a new webhook target.

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

## Tech Stack

| Component       | Technology                          |
|-----------------|-------------------------------------|
| Language        | Go 1.22+                            |
| HTTP router     | [chi](https://github.com/go-chi/chi) |
| Database        | SQLite (default) / PostgreSQL       |
| Messaging       | NATS (embedded or external)         |
| Configuration   | [Viper](https://github.com/spf13/viper) |
| Phone parsing   | Internal `pkg/phone` (E.164 normalization) |
| Container image | `ghcr.io/prenansantana/waid`        |

---

## License

MIT — see [LICENSE](LICENSE).
