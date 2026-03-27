from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass
class Contact:
    id: str
    phone: str
    name: str
    status: str
    created_at: str
    updated_at: str
    bsuid: str | None = None
    external_id: str | None = None
    metadata: dict[str, Any] | None = None
    deleted_at: str | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "Contact":
        return cls(
            id=data["id"],
            phone=data["phone"],
            name=data["name"],
            status=data["status"],
            created_at=data["created_at"],
            updated_at=data["updated_at"],
            bsuid=data.get("bsuid"),
            external_id=data.get("external_id"),
            metadata=data.get("metadata"),
            deleted_at=data.get("deleted_at"),
        )


@dataclass
class IdentityResult:
    match_type: str
    confidence: float
    resolved_at: str
    contact: Contact | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "IdentityResult":
        contact_data = data.get("contact")
        return cls(
            match_type=data["match_type"],
            confidence=data["confidence"],
            resolved_at=data["resolved_at"],
            contact=Contact.from_dict(contact_data) if contact_data else None,
        )


@dataclass
class ImportRowError:
    row: int
    phone: str
    reason: str

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "ImportRowError":
        return cls(
            row=data["row"],
            phone=data["phone"],
            reason=data["reason"],
        )


@dataclass
class ImportReport:
    total: int
    created: int
    updated: int
    errors: int
    details: list[ImportRowError] = field(default_factory=list)

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "ImportReport":
        return cls(
            total=data["total"],
            created=data["created"],
            updated=data["updated"],
            errors=data["errors"],
            details=[ImportRowError.from_dict(e) for e in data.get("details", [])],
        )


@dataclass
class WebhookTarget:
    id: str
    url: str
    events: list[str]
    active: bool
    created_at: str
    secret: str | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "WebhookTarget":
        return cls(
            id=data["id"],
            url=data["url"],
            events=data["events"],
            active=data["active"],
            created_at=data["created_at"],
            secret=data.get("secret"),
        )


@dataclass
class HealthStatus:
    status: str
    database: str
    version: str

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "HealthStatus":
        return cls(
            status=data["status"],
            database=data["database"],
            version=data["version"],
        )


@dataclass
class PaginatedContacts:
    data: list[Contact]
    total: int
    page: int
    per_page: int

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "PaginatedContacts":
        return cls(
            data=[Contact.from_dict(c) for c in data["data"]],
            total=data["total"],
            page=data["page"],
            per_page=data["per_page"],
        )
