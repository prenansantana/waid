ALTER TABLE contacts ADD COLUMN whatsapp_id TEXT;
CREATE INDEX IF NOT EXISTS idx_contacts_whatsapp_id ON contacts(whatsapp_id);
