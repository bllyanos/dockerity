CREATE TABLE apps (
    id                INTEGER PRIMARY KEY,
    name              TEXT NOT NULL UNIQUE,
    repo_url          TEXT NOT NULL,
    default_branch    TEXT NOT NULL,
    deploy_public_key TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL
);
