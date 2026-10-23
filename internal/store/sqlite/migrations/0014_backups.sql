ALTER TABLE environments ADD COLUMN backup_retention INTEGER NOT NULL DEFAULT 5;

-- Backups outlive their environment, so env_id becomes NULL on delete and the
-- names are kept for display.
CREATE TABLE backups (
    id         INTEGER PRIMARY KEY,
    env_id     INTEGER REFERENCES environments (id) ON DELETE SET NULL,
    app_name   TEXT    NOT NULL,
    env_name   TEXT    NOT NULL,
    service    TEXT    NOT NULL,
    volume     TEXT    NOT NULL,
    file       TEXT    NOT NULL UNIQUE,
    size_bytes INTEGER NOT NULL,
    reason     TEXT    NOT NULL,
    created_at TEXT    NOT NULL
);

CREATE INDEX backups_env_id ON backups (env_id, id);
CREATE INDEX backups_volume ON backups (volume, id);
