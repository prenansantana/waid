# waid-sdk

Python SDK for [WAID](https://github.com/prenansantana/waid) — WhatsApp Identity Resolver.

## Installation

```bash
pip install waid-sdk
```

Requires Python 3.10+. The only runtime dependency is [httpx](https://www.python-httpx.org/).

## Quick start

### Synchronous

```python
from waid import WAIDClient

client = WAIDClient(base_url="http://localhost:8080", api_key="your-api-key")

# Resolve a phone number or BSUID
result = client.resolve("+5511999990000")
print(result.match_type, result.contact)

# Create a contact
contact = client.create_contact(phone="+5511999990000", name="Alice", external_id="crm-42")
print(contact.id)

client.close()
```

Use as a context manager to close automatically:

```python
with WAIDClient(base_url="http://localhost:8080", api_key="your-api-key") as client:
    status = client.health()
    print(status.status)  # "ok"
```

### Asynchronous

```python
import asyncio
from waid import AsyncWAIDClient

async def main():
    async with AsyncWAIDClient(base_url="http://localhost:8080", api_key="your-api-key") as client:
        result = await client.resolve("+5511999990000")
        print(result.match_type)

asyncio.run(main())
```

## API reference

Both `WAIDClient` (sync) and `AsyncWAIDClient` (async) expose the same methods. Async versions are prefixed with `await`.

### Constructor

```python
WAIDClient(base_url: str, api_key: str | None = None)
AsyncWAIDClient(base_url: str, api_key: str | None = None)
```

### Health

```python
status: HealthStatus = client.health()
# status.status    → "ok"
# status.database  → "ok" | "error"
# status.version   → str
```

### Resolve

```python
result: IdentityResult = client.resolve(phone_or_id: str)
# result.match_type  → "phone" | "bsuid" | "created" | "not_found"
# result.confidence  → float (0–1)
# result.resolved_at → str (ISO 8601)
# result.contact     → Contact | None
```

### Contacts

```python
# Create
contact: Contact = client.create_contact(
    phone: str,
    name: str,
    external_id: str | None = None,
    metadata: dict | None = None,
)

# Read
contact: Contact = client.get_contact(contact_id: str)

# Update (partial — only provided fields are changed)
contact: Contact = client.update_contact(
    contact_id: str,
    name: str | None = None,
    external_id: str | None = None,
    status: str | None = None,
    metadata: dict | None = None,
)

# Delete (soft)
client.delete_contact(contact_id: str)

# List (paginated)
page: PaginatedContacts = client.list_contacts(
    page: int = 1,
    per_page: int = 50,
    q: str | None = None,   # search query matched against name and phone
)
# page.data     → list[Contact]
# page.total    → int
# page.page     → int
# page.per_page → int
```

### Import

```python
import io

with open("contacts.csv", "rb") as f:
    report: ImportReport = client.import_contacts(f, filename="contacts.csv")

# report.total   → int
# report.created → int
# report.updated → int
# report.errors  → int
# report.details → list[ImportError]
```

### Webhooks

```python
# Register
webhook: WebhookTarget = client.create_webhook(
    url: str,
    events: list[str] | None = None,  # e.g. ["contact.resolved", "contact.created"]
    secret: str | None = None,        # HMAC signing secret
)

# List
webhooks: list[WebhookTarget] = client.list_webhooks()

# Delete
client.delete_webhook(webhook_id: str)
```

### Error handling

All non-2xx responses raise `WAIDError`:

```python
from waid import WAIDClient, WAIDError

client = WAIDClient(base_url="http://localhost:8080")

try:
    contact = client.get_contact("nonexistent-uuid")
except WAIDError as e:
    print(e.status_code)  # 404
    print(e.message)      # "contact not found"
    print(e.body)         # raw parsed response body
```

## Models

| Model | Fields |
|-------|--------|
| `Contact` | `id`, `phone`, `name`, `status`, `created_at`, `updated_at`, `bsuid?`, `external_id?`, `metadata?`, `deleted_at?` |
| `IdentityResult` | `match_type`, `confidence`, `resolved_at`, `contact?` |
| `ImportReport` | `total`, `created`, `updated`, `errors`, `details` |
| `ImportError` | `row`, `phone`, `reason` |
| `WebhookTarget` | `id`, `url`, `events`, `active`, `created_at`, `secret?` |
| `HealthStatus` | `status`, `database`, `version` |
| `PaginatedContacts` | `data`, `total`, `page`, `per_page` |

## Development

```bash
pip install -e ".[dev]"
pytest
```
