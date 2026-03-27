# waid-sdk-go

Go SDK for the [WAID](https://github.com/prenansantana/waid) WhatsApp Identity Resolver API.

Zero external dependencies — stdlib only.

## Install

```sh
go get github.com/prenansantana/waid-sdk-go
```

## Quickstart

```go
package main

import (
    "context"
    "fmt"
    "log"

    waid "github.com/prenansantana/waid-sdk-go"
)

func main() {
    client := waid.NewClient("http://localhost:8080",
        waid.WithAPIKey("your-api-key"),
    )

    result, err := client.Resolve(context.Background(), "+5511999990000")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("match_type=%s contact=%+v\n", result.MatchType, result.Contact)
}
```

## Options

| Option | Description |
|---|---|
| `WithAPIKey(key string)` | Sets the `X-API-Key` header on every request |
| `WithHTTPClient(hc *http.Client)` | Replaces the default `*http.Client` |
| `WithTimeout(d time.Duration)` | Sets a timeout on the default HTTP client |

## Methods

All methods accept `context.Context` as the first argument.

### Identity Resolution

```go
result, err := client.Resolve(ctx, "+5511999990000")
// result: *waid.IdentityResult
```

### Contacts

```go
// Create
contact, err := client.CreateContact(ctx, waid.CreateContactInput{
    Phone: "+5511999990000",
    Name:  "Alice",
})

// Get by ID
contact, err := client.GetContact(ctx, "uuid-here")

// Update (partial — only set fields are changed)
name := "Alice Smith"
contact, err := client.UpdateContact(ctx, "uuid-here", waid.UpdateContactInput{
    Name: &name,
})

// Delete (soft)
err := client.DeleteContact(ctx, "uuid-here")

// List with pagination/search
page, err := client.ListContacts(ctx, waid.ListOpts{
    Page:    1,
    PerPage: 50,
    Query:   "Alice",
})
```

### Bulk Import

```go
f, _ := os.Open("contacts.csv")
defer f.Close()

report, err := client.ImportContacts(ctx, "contacts.csv", f)
fmt.Printf("created=%d updated=%d errors=%d\n", report.Created, report.Updated, report.Errors)
```

### Webhooks

```go
// Register
target, err := client.CreateWebhook(ctx, waid.CreateWebhookInput{
    URL:    "https://your-service.example/hook",
    Events: []string{"contact.resolved", "contact.created"},
    Secret: "hmac-secret",
})

// List
targets, err := client.ListWebhooks(ctx)

// Delete
err := client.DeleteWebhook(ctx, "webhook-uuid")
```

### Health

```go
status, err := client.Health(ctx)
fmt.Println(status.Status, status.Database, status.Version)
```

## Error Handling

API errors are returned as `*waid.WAIDError`:

```go
result, err := client.GetContact(ctx, "missing-id")
if err != nil {
    var wErr *waid.WAIDError
    if errors.As(err, &wErr) {
        fmt.Printf("HTTP %d: %s\n", wErr.StatusCode, wErr.Message)
    }
}
```

## Types

| Type | Description |
|---|---|
| `Contact` | WhatsApp contact with phone, BSUID, metadata, etc. |
| `IdentityResult` | Resolution outcome: match_type, confidence, resolved contact |
| `ImportReport` | Bulk import summary: total, created, updated, errors |
| `WebhookTarget` | Registered webhook endpoint |
| `HealthStatus` | Service health response |
| `PaginatedContacts` | Paginated contact list with total/page/per_page |
| `CreateContactInput` | Input for creating a contact |
| `UpdateContactInput` | Input for partial contact update |
| `ListOpts` | Pagination/search options for ListContacts |
| `CreateWebhookInput` | Input for registering a webhook target |
| `WAIDError` | API error with StatusCode, Message, and raw Body |
