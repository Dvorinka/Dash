-- +goose Up
-- Uptime monitors and their check history (merge phase M1).
CREATE TABLE monitors (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    type           TEXT NOT NULL CHECK (type IN ('http','tcp','ping','dns','keyword','json','push')),
    url            TEXT NOT NULL DEFAULT '',
    hostname       TEXT NOT NULL DEFAULT '',
    port           INTEGER NOT NULL DEFAULT 0,
    method         TEXT NOT NULL DEFAULT 'GET',
    headers        JSON NOT NULL DEFAULT '{}',
    body           TEXT NOT NULL DEFAULT '',
    keyword        TEXT NOT NULL DEFAULT '',
    keyword_invert INTEGER NOT NULL DEFAULT 0,
    json_query     TEXT NOT NULL DEFAULT '',
    expected       TEXT NOT NULL DEFAULT '',
    dns_type       TEXT NOT NULL DEFAULT 'A',
    interval_s     INTEGER NOT NULL DEFAULT 60,
    timeout_s      INTEGER NOT NULL DEFAULT 10,
    retries        INTEGER NOT NULL DEFAULT 0,
    active         INTEGER NOT NULL DEFAULT 1,
    status         TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('up','down','pending','paused','maintenance')),
    push_token     TEXT NOT NULL DEFAULT '',
    tags           JSON NOT NULL DEFAULT '[]',
    notes          TEXT NOT NULL DEFAULT '',
    position       REAL NOT NULL DEFAULT 0,
    last_check     TEXT,
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX monitors_due ON monitors(active, last_check);

CREATE TABLE heartbeats (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id  TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status      TEXT NOT NULL CHECK (status IN ('up','down')),
    ping_ms     INTEGER NOT NULL DEFAULT 0,
    msg         TEXT NOT NULL DEFAULT '',
    cert_expiry INTEGER NOT NULL DEFAULT -1,
    checked_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX heartbeats_monitor_time ON heartbeats(monitor_id, checked_at);

-- +goose Down
DROP TABLE heartbeats;
DROP TABLE monitors;
