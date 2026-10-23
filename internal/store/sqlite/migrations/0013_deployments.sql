CREATE TABLE deployments (
    id             INTEGER PRIMARY KEY,
    env_id         INTEGER NOT NULL REFERENCES environments (id) ON DELETE CASCADE,
    build_id       INTEGER NOT NULL REFERENCES builds (id),
    spec_version   INTEGER NOT NULL,
    secret_version INTEGER NOT NULL DEFAULT 0,
    status         TEXT    NOT NULL,
    current        INTEGER NOT NULL DEFAULT 0,
    triggered_by   INTEGER REFERENCES users (id) ON DELETE SET NULL,
    error          TEXT    NOT NULL DEFAULT '',
    created_at     TEXT    NOT NULL,
    started_at     TEXT,
    finished_at    TEXT
);

CREATE INDEX deployments_env_id ON deployments (env_id, id);

-- The environment lock: at most one pending or running deployment per environment.
CREATE UNIQUE INDEX deployments_active ON deployments (env_id) WHERE status IN ('pending', 'running');

-- At most one current deployment per environment.
CREATE UNIQUE INDEX deployments_current ON deployments (env_id) WHERE current = 1;

CREATE TABLE deployment_steps (
    id            INTEGER PRIMARY KEY,
    deployment_id INTEGER NOT NULL REFERENCES deployments (id) ON DELETE CASCADE,
    position      INTEGER NOT NULL,
    kind          TEXT    NOT NULL,
    service       TEXT    NOT NULL DEFAULT '',
    status        TEXT    NOT NULL,
    exit_code     INTEGER,
    message       TEXT    NOT NULL DEFAULT '',
    started_at    TEXT,
    finished_at   TEXT,
    UNIQUE (deployment_id, position)
);
