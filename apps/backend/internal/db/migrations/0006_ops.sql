-- +goose Up
CREATE TABLE incidents (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    severity   TEXT NOT NULL DEFAULT 'major',   -- minor | major | critical
    status     TEXT NOT NULL DEFAULT 'open',    -- open | ack | resolved | closed
    monitor_id TEXT REFERENCES monitors(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_incidents_status ON incidents(status);

CREATE TABLE incident_updates (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    status      TEXT NOT NULL,
    message     TEXT NOT NULL DEFAULT '',
    at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_incident_updates_inc ON incident_updates(incident_id);

CREATE TABLE maintenance_windows (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    starts_at   TEXT NOT NULL,
    ends_at     TEXT NOT NULL,
    monitor_ids TEXT NOT NULL DEFAULT '[]',  -- JSON array; empty covers all
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE status_pages (
    id          TEXT PRIMARY KEY,
    slug        TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    monitor_ids TEXT NOT NULL DEFAULT '[]',  -- JSON array; empty = all monitors
    system_ids  TEXT NOT NULL DEFAULT '[]',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- +goose Down
DROP TABLE IF EXISTS status_pages;
DROP TABLE IF EXISTS maintenance_windows;
DROP TABLE IF EXISTS incident_updates;
DROP TABLE IF EXISTS incidents;
