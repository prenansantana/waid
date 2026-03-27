from __future__ import annotations

from typing import Any


class WAIDError(Exception):
    """Raised when the WAID API returns a non-2xx response."""

    def __init__(self, status_code: int, message: str, body: Any = None) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.message = message
        self.body = body

    def __repr__(self) -> str:
        return f"WAIDError(status_code={self.status_code!r}, message={self.message!r})"
