CREATE TABLE environments (
    id         INTEGER PRIMARY KEY,
    app_id     INTEGER NOT NULL REFERENCES apps (id) ON DELETE CASCADE,
    name       TEXT    NOT NULL,
    type       TEXT    NOT NULL,
    domain     TEXT    NOT NULL DEFAULT '',
    created_at TEXT    NOT NULL,
    updated_at TEXT    NOT NULL,
    UNIQUE (app_id, name)
);

-- Two environments can never claim the same domain.
CREATE UNIQUE INDEX environments_domain ON environments (domain) WHERE domain <> '';
