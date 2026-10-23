-- Ephemeral environments expire; their TTL is extended on every deployment.
ALTER TABLE environments ADD COLUMN ttl_seconds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE environments ADD COLUMN expires_at TEXT;
CREATE INDEX environments_expires_at ON environments (expires_at) WHERE expires_at IS NOT NULL;
