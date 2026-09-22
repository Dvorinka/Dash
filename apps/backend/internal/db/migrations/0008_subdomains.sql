-- +goose Up
CREATE TABLE subdomains (
    id         TEXT PRIMARY KEY,
    domain_id  TEXT NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    ips        JSON NOT NULL DEFAULT '[]',
    first_seen TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    last_seen  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    UNIQUE(domain_id, name)
);
CREATE INDEX subdomains_domain_idx ON subdomains(domain_id);

-- Discovery runs at most once per day per domain, inside the domain sweep.
ALTER TABLE domains ADD COLUMN sub_checked_at TEXT;

-- +goose Down
ALTER TABLE domains DROP COLUMN sub_checked_at;
DROP TABLE IF EXISTS subdomains;
