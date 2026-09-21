-- +goose Up
CREATE TABLE systems (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    token        TEXT NOT NULL UNIQUE,
    host         TEXT NOT NULL DEFAULT '',
    os           TEXT NOT NULL DEFAULT '',
    arch         TEXT NOT NULL DEFAULT '',
    cpu_model    TEXT NOT NULL DEFAULT '',
    cores        INTEGER NOT NULL DEFAULT 0,
    interval_s   INTEGER NOT NULL DEFAULT 10,
    status       TEXT NOT NULL DEFAULT 'pending',
    last_seen_at TEXT,
    latest       TEXT NOT NULL DEFAULT '{}',
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- Whole agent payload per tick; no sub-field queries, so one JSON column
-- instead of a wide row that drifts with every agent version.
CREATE TABLE system_stats (
    system_id TEXT NOT NULL REFERENCES systems(id) ON DELETE CASCADE,
    ts        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    payload   TEXT NOT NULL,
    PRIMARY KEY (system_id, ts)
);

CREATE INDEX idx_system_stats_ts ON system_stats(ts);

-- +goose Down
DROP TABLE IF EXISTS system_stats;
DROP TABLE IF EXISTS systems;
