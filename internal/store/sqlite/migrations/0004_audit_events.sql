CREATE TABLE audit_events (
    id             INTEGER PRIMARY KEY,
    at             TEXT    NOT NULL,
    actor_user_id  INTEGER REFERENCES users (id) ON DELETE SET NULL,
    actor_token_id INTEGER,
    action         TEXT    NOT NULL,
    target_type    TEXT    NOT NULL DEFAULT '',
    target_id      TEXT    NOT NULL DEFAULT '',
    metadata       TEXT    NOT NULL DEFAULT '{}',
    remote_addr    TEXT    NOT NULL DEFAULT ''
);
