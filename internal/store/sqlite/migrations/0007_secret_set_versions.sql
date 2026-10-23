-- Immutable, encrypted secret set versions. Key names are stored in clear so
-- versions can be listed without decrypting; values only exist in ciphertext.
CREATE TABLE secret_set_versions (
    scope      TEXT    NOT NULL,
    version    INTEGER NOT NULL,
    key_names  TEXT    NOT NULL,
    ciphertext BLOB    NOT NULL,
    nonce      BLOB    NOT NULL,
    created_at TEXT    NOT NULL,
    created_by INTEGER REFERENCES users (id) ON DELETE SET NULL,
    PRIMARY KEY (scope, version)
);
