-- SSH host keys pinned on first use for git remotes.
CREATE TABLE known_hosts (
    host        TEXT PRIMARY KEY,
    key_type    TEXT NOT NULL,
    key         TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    added_at    TEXT NOT NULL
);
