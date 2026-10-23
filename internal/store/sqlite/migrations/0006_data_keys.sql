-- Data keys wrapped by the master key. The master key itself is never stored.
CREATE TABLE data_keys (
    scope         TEXT PRIMARY KEY,
    wrapped_key   BLOB NOT NULL,
    nonce         BLOB NOT NULL,
    master_key_id TEXT NOT NULL,
    created_at    TEXT NOT NULL
);
