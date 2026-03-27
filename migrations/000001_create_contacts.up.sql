CREATE TABLE IF NOT EXISTS contacts (
    id TEXT PRIMARY KEY,
    phone TEXT NOT NULL UNIQUE,
    bsuid TEXT UNIQUE,
    external_id TEXT,
    name TEXT NOT NULL DEFAULT '',
    metadata TEXT DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_contacts_external_id ON contacts(external_id);
CREATE INDEX IF NOT EXISTS idx_contacts_status ON contacts(status);
CREATE INDEX IF NOT EXISTS idx_contacts_phone_status ON contacts(phone, status) WHERE deleted_at IS NULL;
