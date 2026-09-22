-- +goose Up
-- Opt-in local auth. Empty users table = setup mode; auth_enabled settings
-- key gates the middleware. Sessions live in SQLite — restart-safe, and a
-- `dash-agent`-style bearer token path is unaffected.
CREATE TABLE users (
    id         TEXT PRIMARY KEY,
    username   TEXT NOT NULL UNIQUE,
    pass_hash  TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    expires_at TEXT NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions(expires_at);

-- +goose Down
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
