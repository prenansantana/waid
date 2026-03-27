from __future__ import annotations

import urllib.parse
from typing import Any, BinaryIO

import httpx

from .errors import WAIDError
from .models import (
    Contact,
    HealthStatus,
    IdentityResult,
    ImportReport,
    PaginatedContacts,
    WebhookTarget,
)


def _raise_for_status(response: httpx.Response) -> None:
    if response.is_success:
        return
    try:
        body = response.json()
        message = body.get("error", response.text)
    except Exception:
        body = response.text
        message = response.text
    raise WAIDError(status_code=response.status_code, message=message, body=body)


class WAIDClient:
    """Synchronous WAID client backed by httpx.Client."""

    def __init__(self, base_url: str, api_key: str | None = None) -> None:
        headers: dict[str, str] = {"Accept": "application/json"}
        if api_key:
            headers["X-API-Key"] = api_key
        self._http = httpx.Client(base_url=base_url.rstrip("/"), headers=headers)

    def close(self) -> None:
        self._http.close()

    def __enter__(self) -> "WAIDClient":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()

    # ------------------------------------------------------------------
    # Health
    # ------------------------------------------------------------------

    def health(self) -> HealthStatus:
        """Return service health status."""
        resp = self._http.get("/health")
        _raise_for_status(resp)
        return HealthStatus.from_dict(resp.json())

    # ------------------------------------------------------------------
    # Resolve
    # ------------------------------------------------------------------

    def resolve(self, phone_or_id: str) -> IdentityResult:
        """Resolve a phone number or BSUID to a contact."""
        resp = self._http.get(f"/resolve/{urllib.parse.quote(phone_or_id, safe='')}")
        _raise_for_status(resp)
        return IdentityResult.from_dict(resp.json())

    # ------------------------------------------------------------------
    # Contacts
    # ------------------------------------------------------------------

    def create_contact(
        self,
        phone: str,
        name: str,
        external_id: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Contact:
        """Create a new contact."""
        payload: dict[str, Any] = {"phone": phone, "name": name}
        if external_id is not None:
            payload["external_id"] = external_id
        if metadata is not None:
            payload["metadata"] = metadata
        resp = self._http.post("/contacts", json=payload)
        _raise_for_status(resp)
        return Contact.from_dict(resp.json())

    def get_contact(self, contact_id: str) -> Contact:
        """Get a contact by UUID."""
        resp = self._http.get(f"/contacts/{urllib.parse.quote(contact_id, safe='')}")
        _raise_for_status(resp)
        return Contact.from_dict(resp.json())

    def update_contact(
        self,
        contact_id: str,
        name: str | None = None,
        external_id: str | None = None,
        status: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Contact:
        """Partial update of a contact (only provided fields are changed)."""
        payload: dict[str, Any] = {}
        if name is not None:
            payload["name"] = name
        if external_id is not None:
            payload["external_id"] = external_id
        if status is not None:
            payload["status"] = status
        if metadata is not None:
            payload["metadata"] = metadata
        resp = self._http.put(f"/contacts/{urllib.parse.quote(contact_id, safe='')}", json=payload)
        _raise_for_status(resp)
        return Contact.from_dict(resp.json())

    def delete_contact(self, contact_id: str) -> None:
        """Soft-delete a contact."""
        resp = self._http.delete(f"/contacts/{urllib.parse.quote(contact_id, safe='')}")
        _raise_for_status(resp)

    def list_contacts(
        self,
        page: int = 1,
        per_page: int = 50,
        q: str | None = None,
    ) -> PaginatedContacts:
        """List contacts with optional pagination and search."""
        params: dict[str, Any] = {"page": page, "per_page": per_page}
        if q is not None:
            params["q"] = q
        resp = self._http.get("/contacts", params=params)
        _raise_for_status(resp)
        return PaginatedContacts.from_dict(resp.json())

    def import_contacts(self, file: BinaryIO, filename: str = "contacts.csv") -> ImportReport:
        """Bulk-upsert contacts from a CSV or JSON file."""
        resp = self._http.post("/import", files={"file": (filename, file)})
        _raise_for_status(resp)
        return ImportReport.from_dict(resp.json())

    # ------------------------------------------------------------------
    # Webhooks
    # ------------------------------------------------------------------

    def create_webhook(
        self,
        url: str,
        events: list[str] | None = None,
        secret: str | None = None,
    ) -> WebhookTarget:
        """Register a webhook target."""
        payload: dict[str, Any] = {"url": url}
        if events is not None:
            payload["events"] = events
        if secret is not None:
            payload["secret"] = secret
        resp = self._http.post("/webhooks", json=payload)
        _raise_for_status(resp)
        return WebhookTarget.from_dict(resp.json())

    def list_webhooks(self) -> list[WebhookTarget]:
        """List all active webhook targets."""
        resp = self._http.get("/webhooks")
        _raise_for_status(resp)
        return [WebhookTarget.from_dict(w) for w in resp.json()]

    def delete_webhook(self, webhook_id: str) -> None:
        """Remove a webhook target."""
        resp = self._http.delete(f"/webhooks/{urllib.parse.quote(webhook_id, safe='')}")
        _raise_for_status(resp)


class AsyncWAIDClient:
    """Asynchronous WAID client backed by httpx.AsyncClient."""

    def __init__(self, base_url: str, api_key: str | None = None) -> None:
        headers: dict[str, str] = {"Accept": "application/json"}
        if api_key:
            headers["X-API-Key"] = api_key
        self._http = httpx.AsyncClient(base_url=base_url.rstrip("/"), headers=headers)

    async def aclose(self) -> None:
        await self._http.aclose()

    async def __aenter__(self) -> "AsyncWAIDClient":
        return self

    async def __aexit__(self, *args: Any) -> None:
        await self.aclose()

    # ------------------------------------------------------------------
    # Health
    # ------------------------------------------------------------------

    async def health(self) -> HealthStatus:
        """Return service health status."""
        resp = await self._http.get("/health")
        _raise_for_status(resp)
        return HealthStatus.from_dict(resp.json())

    # ------------------------------------------------------------------
    # Resolve
    # ------------------------------------------------------------------

    async def resolve(self, phone_or_id: str) -> IdentityResult:
        """Resolve a phone number or BSUID to a contact."""
        resp = await self._http.get(f"/resolve/{urllib.parse.quote(phone_or_id, safe='')}")
        _raise_for_status(resp)
        return IdentityResult.from_dict(resp.json())

    # ------------------------------------------------------------------
    # Contacts
    # ------------------------------------------------------------------

    async def create_contact(
        self,
        phone: str,
        name: str,
        external_id: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Contact:
        """Create a new contact."""
        payload: dict[str, Any] = {"phone": phone, "name": name}
        if external_id is not None:
            payload["external_id"] = external_id
        if metadata is not None:
            payload["metadata"] = metadata
        resp = await self._http.post("/contacts", json=payload)
        _raise_for_status(resp)
        return Contact.from_dict(resp.json())

    async def get_contact(self, contact_id: str) -> Contact:
        """Get a contact by UUID."""
        resp = await self._http.get(f"/contacts/{urllib.parse.quote(contact_id, safe='')}")
        _raise_for_status(resp)
        return Contact.from_dict(resp.json())

    async def update_contact(
        self,
        contact_id: str,
        name: str | None = None,
        external_id: str | None = None,
        status: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Contact:
        """Partial update of a contact."""
        payload: dict[str, Any] = {}
        if name is not None:
            payload["name"] = name
        if external_id is not None:
            payload["external_id"] = external_id
        if status is not None:
            payload["status"] = status
        if metadata is not None:
            payload["metadata"] = metadata
        resp = await self._http.put(f"/contacts/{urllib.parse.quote(contact_id, safe='')}", json=payload)
        _raise_for_status(resp)
        return Contact.from_dict(resp.json())

    async def delete_contact(self, contact_id: str) -> None:
        """Soft-delete a contact."""
        resp = await self._http.delete(f"/contacts/{urllib.parse.quote(contact_id, safe='')}")
        _raise_for_status(resp)

    async def list_contacts(
        self,
        page: int = 1,
        per_page: int = 50,
        q: str | None = None,
    ) -> PaginatedContacts:
        """List contacts with optional pagination and search."""
        params: dict[str, Any] = {"page": page, "per_page": per_page}
        if q is not None:
            params["q"] = q
        resp = await self._http.get("/contacts", params=params)
        _raise_for_status(resp)
        return PaginatedContacts.from_dict(resp.json())

    async def import_contacts(self, file: BinaryIO, filename: str = "contacts.csv") -> ImportReport:
        """Bulk-upsert contacts from a CSV or JSON file."""
        resp = await self._http.post("/import", files={"file": (filename, file)})
        _raise_for_status(resp)
        return ImportReport.from_dict(resp.json())

    # ------------------------------------------------------------------
    # Webhooks
    # ------------------------------------------------------------------

    async def create_webhook(
        self,
        url: str,
        events: list[str] | None = None,
        secret: str | None = None,
    ) -> WebhookTarget:
        """Register a webhook target."""
        payload: dict[str, Any] = {"url": url}
        if events is not None:
            payload["events"] = events
        if secret is not None:
            payload["secret"] = secret
        resp = await self._http.post("/webhooks", json=payload)
        _raise_for_status(resp)
        return WebhookTarget.from_dict(resp.json())

    async def list_webhooks(self) -> list[WebhookTarget]:
        """List all active webhook targets."""
        resp = await self._http.get("/webhooks")
        _raise_for_status(resp)
        return [WebhookTarget.from_dict(w) for w in resp.json()]

    async def delete_webhook(self, webhook_id: str) -> None:
        """Remove a webhook target."""
        resp = await self._http.delete(f"/webhooks/{urllib.parse.quote(webhook_id, safe='')}")
        _raise_for_status(resp)
