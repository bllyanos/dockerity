CREATE TABLE builds (
    id           INTEGER PRIMARY KEY,
    app_id       INTEGER NOT NULL REFERENCES apps (id) ON DELETE CASCADE,
    ref          TEXT    NOT NULL,
    commit_sha   TEXT    NOT NULL DEFAULT '',
    status       TEXT    NOT NULL,
    triggered_by INTEGER REFERENCES users (id) ON DELETE SET NULL,
    error        TEXT    NOT NULL DEFAULT '',
    created_at   TEXT    NOT NULL,
    started_at   TEXT,
    finished_at  TEXT
);

CREATE INDEX builds_app_id ON builds (app_id, id);
CREATE INDEX builds_status ON builds (status);

CREATE TABLE build_images (
    build_id  INTEGER NOT NULL REFERENCES builds (id) ON DELETE CASCADE,
    service   TEXT    NOT NULL,
    image_tag TEXT    NOT NULL,
    image_id  TEXT    NOT NULL,
    PRIMARY KEY (build_id, service)
);
