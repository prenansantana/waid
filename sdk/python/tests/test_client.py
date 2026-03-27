from __future__ import annotations

import json
from typing import Any

import httpx
import pytest

from waid import (
    AsyncWAIDClient,
    WAIDClient,
    WAIDError,
)
from waid.models import (
    Contact,
    HealthStatus,
    IdentityResult,
    ImportReport,
    PaginatedContacts,
    WebhookTarget,
)

# ---------------------------------------------------------------------------
# Fixture helpers
# ---------------------------------------------------------------------------

CONTACT_PAYLOAD: dict[str, Any] = {
    "id": "00000000-0000-0000-0000-000000000001",
    "phone": "+5511999990000",
    "name": "Alice",
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "bsuid": None,
    "external_id": "crm-42",
    "metadata": {"tag": "vip"},
    "deleted_at": None,
}

IDENTITY_RESULT_PAYLOAD: dict[str, Any] = {
    "match_type": "phone",
    "confidence": 1.0,
    "resolved_at": "2024-01-01T00:00:00Z",
    "contact": CONTACT_PAYLOAD,
}

PAGINATED_PAYLOAD: dict[str, Any] = {
    "data": [CONTACT_PAYLOAD],
    "total": 1,
    "page": 1,
    "per_page": 50,
}

WEBHOOK_PAYLOAD: dict[str, Any] = {
    "id": "00000000-0000-0000-0000-000000000002",
    "url": "https://example.com/hook",
    "events": ["contact.resolved"],
    "active": True,
    "created_at": "2024-01-01T00:00:00Z",
}

IMPORT_REPORT_PAYLOAD: dict[str, Any] = {
    "total": 10,
    "created": 8,
    "updated": 2,
    "errors": 0,
    "details": [],
}

HEALTH_PAYLOAD: dict[str, Any] = {
    "status": "ok",
    "database": "ok",
    "version": "dev",
}


def _response(status: int, body: Any) -> httpx.Response:
    content = json.dumps(body).encode()
    return httpx.Response(status, content=content, headers={"content-type": "application/json"})


def _empty_response(status: int) -> httpx.Response:
    return httpx.Response(status, content=b"")


# ---------------------------------------------------------------------------
# Sync client tests
# ---------------------------------------------------------------------------


def make_sync_client(handler: Any) -> WAIDClient:
    transport = httpx.MockTransport(handler)
    client = WAIDClient.__new__(WAIDClient)
    client._http = httpx.Client(base_url="http://test", transport=transport)
    return client


def test_health():
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "GET"
        assert request.url.path == "/health"
        return _response(200, HEALTH_PAYLOAD)

    client = make_sync_client(handler)
    result = client.health()
    assert isinstance(result, HealthStatus)
    assert result.status == "ok"
    assert result.database == "ok"
    assert result.version == "dev"


def test_resolve():
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/resolve/+5511999990000"
        return _response(200, IDENTITY_RESULT_PAYLOAD)

    client = make_sync_client(handler)
    result = client.resolve("+5511999990000")
    assert isinstance(result, IdentityResult)
    assert result.match_type == "phone"
    assert result.confidence == 1.0
    assert isinstance(result.contact, Contact)
    assert result.contact.phone == "+5511999990000"


def test_create_contact():
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "POST"
        assert request.url.path == "/contacts"
        body = json.loads(request.content)
        assert body["phone"] == "+5511999990000"
        assert body["name"] == "Alice"
        return _response(201, CONTACT_PAYLOAD)

    client = make_sync_client(handler)
    contact = client.create_contact(phone="+5511999990000", name="Alice")
    assert isinstance(contact, Contact)
    assert contact.id == CONTACT_PAYLOAD["id"]


def test_get_contact():
    cid = CONTACT_PAYLOAD["id"]

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == f"/contacts/{cid}"
        return _response(200, CONTACT_PAYLOAD)

    client = make_sync_client(handler)
    contact = client.get_contact(cid)
    assert contact.name == "Alice"


def test_update_contact():
    cid = CONTACT_PAYLOAD["id"]

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "PUT"
        body = json.loads(request.content)
        assert body["name"] == "Alice Smith"
        updated = {**CONTACT_PAYLOAD, "name": "Alice Smith"}
        return _response(200, updated)

    client = make_sync_client(handler)
    contact = client.update_contact(cid, name="Alice Smith")
    assert contact.name == "Alice Smith"


def test_delete_contact():
    cid = CONTACT_PAYLOAD["id"]

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "DELETE"
        return _empty_response(204)

    client = make_sync_client(handler)
    client.delete_contact(cid)  # should not raise


def test_list_contacts():
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/contacts"
        assert request.url.params["page"] == "1"
        return _response(200, PAGINATED_PAYLOAD)

    client = make_sync_client(handler)
    result = client.list_contacts()
    assert isinstance(result, PaginatedContacts)
    assert result.total == 1
    assert len(result.data) == 1


def test_list_contacts_with_search():
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.params["q"] == "Alice"
        return _response(200, PAGINATED_PAYLOAD)

    client = make_sync_client(handler)
    result = client.list_contacts(q="Alice")
    assert result.total == 1


def test_import_contacts():
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "POST"
        assert request.url.path == "/import"
        return _response(200, IMPORT_REPORT_PAYLOAD)

    import io
    client = make_sync_client(handler)
    f = io.BytesIO(b"phone,name\n+5511999990000,Alice")
    report = client.import_contacts(f, filename="contacts.csv")
    assert isinstance(report, ImportReport)
    assert report.created == 8


def test_create_webhook():
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "POST"
        body = json.loads(request.content)
        assert body["url"] == "https://example.com/hook"
        return _response(201, WEBHOOK_PAYLOAD)

    client = make_sync_client(handler)
    wh = client.create_webhook(url="https://example.com/hook", events=["contact.resolved"])
    assert isinstance(wh, WebhookTarget)
    assert wh.active is True


def test_list_webhooks():
    def handler(request: httpx.Request) -> httpx.Response:
        return _response(200, [WEBHOOK_PAYLOAD])

    client = make_sync_client(handler)
    webhooks = client.list_webhooks()
    assert len(webhooks) == 1
    assert webhooks[0].id == WEBHOOK_PAYLOAD["id"]


def test_delete_webhook():
    wid = WEBHOOK_PAYLOAD["id"]

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "DELETE"
        assert wid in request.url.path
        return _empty_response(204)

    client = make_sync_client(handler)
    client.delete_webhook(wid)  # should not raise


def test_error_raises_waid_error():
    def handler(request: httpx.Request) -> httpx.Response:
        return _response(404, {"error": "contact not found"})

    client = make_sync_client(handler)
    with pytest.raises(WAIDError) as exc_info:
        client.get_contact("nonexistent-id")

    err = exc_info.value
    assert err.status_code == 404
    assert "contact not found" in err.message


def test_api_key_header():
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.headers.get("X-API-Key") == "secret-key"
        return _response(200, HEALTH_PAYLOAD)

    transport = httpx.MockTransport(handler)
    client = WAIDClient.__new__(WAIDClient)
    client._http = httpx.Client(
        base_url="http://test",
        transport=transport,
        headers={"X-API-Key": "secret-key", "Accept": "application/json"},
    )
    client.health()


# ---------------------------------------------------------------------------
# Async client tests
# ---------------------------------------------------------------------------


def make_async_client(handler: Any) -> AsyncWAIDClient:
    transport = httpx.MockTransport(handler)
    client = AsyncWAIDClient.__new__(AsyncWAIDClient)
    client._http = httpx.AsyncClient(base_url="http://test", transport=transport)
    return client


@pytest.mark.asyncio
async def test_async_health():
    def handler(request: httpx.Request) -> httpx.Response:
        return _response(200, HEALTH_PAYLOAD)

    client = make_async_client(handler)
    result = await client.health()
    assert result.status == "ok"


@pytest.mark.asyncio
async def test_async_resolve():
    def handler(request: httpx.Request) -> httpx.Response:
        return _response(200, IDENTITY_RESULT_PAYLOAD)

    client = make_async_client(handler)
    result = await client.resolve("+5511999990000")
    assert result.match_type == "phone"


@pytest.mark.asyncio
async def test_async_create_contact():
    def handler(request: httpx.Request) -> httpx.Response:
        return _response(201, CONTACT_PAYLOAD)

    client = make_async_client(handler)
    contact = await client.create_contact(phone="+5511999990000", name="Alice")
    assert contact.name == "Alice"


@pytest.mark.asyncio
async def test_async_list_contacts():
    def handler(request: httpx.Request) -> httpx.Response:
        return _response(200, PAGINATED_PAYLOAD)

    client = make_async_client(handler)
    result = await client.list_contacts()
    assert result.total == 1


@pytest.mark.asyncio
async def test_async_error_raises_waid_error():
    def handler(request: httpx.Request) -> httpx.Response:
        return _response(500, {"error": "internal server error"})

    client = make_async_client(handler)
    with pytest.raises(WAIDError) as exc_info:
        await client.health()

    assert exc_info.value.status_code == 500
