CREATE TABLE app_spec_versions (
    app_id     INTEGER NOT NULL REFERENCES apps (id) ON DELETE CASCADE,
    version    INTEGER NOT NULL,
    yaml       TEXT    NOT NULL,
    created_at TEXT    NOT NULL,
    created_by INTEGER REFERENCES users (id) ON DELETE SET NULL,
    PRIMARY KEY (app_id, version)
);
