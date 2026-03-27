# @waid/sdk

TypeScript/JavaScript SDK for the [WAID](https://github.com/prenansantana/waid) WhatsApp Identity Resolver API.

Zero external dependencies. Uses native `fetch` (Node 18+).

## Installation

```bash
npm install @waid/sdk
```

## Quickstart

```typescript
import { WAIDClient } from "@waid/sdk";

const client = new WAIDClient({
  baseURL: "http://localhost:8080",
  apiKey: "your-api-key", // optional — only required when server has WAID_SERVER_API_KEY set
});

// Resolve a phone number or BSUID
const result = await client.resolve("+5511999990000");
console.log(result.contact, result.match_type);

// Create a contact
const contact = await client.createContact({
  phone: "+5511999990000",
  name: "Alice",
  external_id: "crm-42",
  metadata: { tag: "vip" },
});
```

## Methods

### `health()`

Check service liveness.

```typescript
const status = await client.health();
// { status: "ok", database: "ok", version: "dev" }
```

### `resolve(phoneOrId)`

Resolve a phone number (E.164) or BSUID to a contact.

```typescript
const result = await client.resolve("+5511999990000");
// result.match_type: "phone" | "bsuid" | "created" | "not_found"
// result.contact: Contact | null
// result.confidence: number (0–1)
```

### `createContact(input)`

Create a new contact.

```typescript
const contact = await client.createContact({
  phone: "+5511999990000", // required — normalized to E.164
  name: "Alice",           // required
  external_id: "crm-42",  // optional
  metadata: { tag: "vip" }, // optional
});
```

### `listContacts(options?)`

List contacts with optional pagination and search.

```typescript
const page = await client.listContacts({ page: 1, per_page: 50, q: "Alice" });
// page.data: Contact[]
// page.total, page.page, page.per_page
```

### `getContact(id)`

Fetch a single contact by UUID.

```typescript
const contact = await client.getContact("550e8400-e29b-41d4-a716-446655440000");
```

### `updateContact(id, input)`

Partially update a contact. Only provided fields are modified.

```typescript
const updated = await client.updateContact(id, {
  name: "Alice Smith",
  status: "active",
  metadata: { tag: "premium" },
});
```

### `deleteContact(id)`

Soft-delete a contact by UUID.

```typescript
await client.deleteContact("550e8400-e29b-41d4-a716-446655440000");
```

### `importContacts(file, filename?)`

Bulk-upsert contacts from a CSV or JSON `Blob`/`File`. Rows are upserted by phone number.

```typescript
import { readFileSync } from "fs";

const blob = new Blob([readFileSync("contacts.csv")], { type: "text/csv" });
const report = await client.importContacts(blob, "contacts.csv");
// report: { total, created, updated, errors, details? }
```

### `createWebhook(input)`

Register an outbound webhook target.

```typescript
const webhook = await client.createWebhook({
  url: "https://your-service.example/hook",
  events: ["contact.resolved", "contact.created"],
  secret: "hmac-signing-secret", // optional
});
```

### `listWebhooks()`

List all active webhook targets.

```typescript
const webhooks = await client.listWebhooks();
```

### `deleteWebhook(id)`

Remove a webhook target by UUID.

```typescript
await client.deleteWebhook("550e8400-e29b-41d4-a716-446655440000");
```

### `inbound(source, payload)`

Forward a raw WhatsApp gateway payload to WAID for normalization and identity resolution.

Supported sources: `waha`, `evolution`, `meta`, `generic`

```typescript
const event = await client.inbound("waha", rawPayload);
// event: InboundEvent
```

## Error Handling

All methods throw `WAIDError` on non-2xx responses.

```typescript
import { WAIDError } from "@waid/sdk";

try {
  await client.getContact("non-existent-id");
} catch (err) {
  if (err instanceof WAIDError) {
    console.error(err.message); // error message from API
    console.error(err.status);  // HTTP status code
    console.error(err.body);    // raw response body
  }
}
```

## Building

```bash
npm run build
```

Output is emitted to `dist/`.

## Requirements

- Node.js 18 or later (native `fetch` required)
- No external dependencies
