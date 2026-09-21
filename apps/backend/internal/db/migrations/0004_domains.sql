-- +goose Up
-- Domain monitoring (merge phase M2). One wide table mirrors the flat
-- lookup Result; domain_checks snapshots each refresh for change history.
CREATE TABLE domains (
    id                 TEXT PRIMARY KEY,
    name               TEXT NOT NULL UNIQUE,
    tld                TEXT NOT NULL DEFAULT '',
    active             INTEGER NOT NULL DEFAULT 1,
    auto_renew         INTEGER NOT NULL DEFAULT 0,
    alert_days_before  INTEGER NOT NULL DEFAULT 30,
    tags               JSON NOT NULL DEFAULT '[]',
    notes              TEXT NOT NULL DEFAULT '',
    -- whois / rdap
    expiry_date        TEXT,
    creation_date      TEXT,
    updated_date       TEXT,
    registrar_name     TEXT NOT NULL DEFAULT '',
    registrar_id       TEXT NOT NULL DEFAULT '',
    registrar_url      TEXT NOT NULL DEFAULT '',
    registry_domain_id TEXT NOT NULL DEFAULT '',
    dnssec             TEXT NOT NULL DEFAULT '',
    statuses           JSON NOT NULL DEFAULT '[]',
    privacy_enabled    INTEGER NOT NULL DEFAULT 0,
    transfer_lock      INTEGER NOT NULL DEFAULT 0,
    registrant_name    TEXT NOT NULL DEFAULT '',
    registrant_org     TEXT NOT NULL DEFAULT '',
    registrant_country TEXT NOT NULL DEFAULT '',
    abuse_email        TEXT NOT NULL DEFAULT '',
    -- dns
    name_servers       JSON NOT NULL DEFAULT '[]',
    mx_records         JSON NOT NULL DEFAULT '[]',
    txt_records        JSON NOT NULL DEFAULT '[]',
    cname              TEXT NOT NULL DEFAULT '',
    ipv4               JSON NOT NULL DEFAULT '[]',
    ipv6               JSON NOT NULL DEFAULT '[]',
    -- tls leaf cert
    ssl_issuer         TEXT NOT NULL DEFAULT '',
    ssl_valid_from     TEXT,
    ssl_valid_to       TEXT,
    ssl_subject        TEXT NOT NULL DEFAULT '',
    ssl_fingerprint    TEXT NOT NULL DEFAULT '',
    ssl_key_size       INTEGER NOT NULL DEFAULT 0,
    ssl_sig_algo       TEXT NOT NULL DEFAULT '',
    ssl_alt_names      JSON NOT NULL DEFAULT '[]',
    -- host geo
    host_country       TEXT NOT NULL DEFAULT '',
    host_country_code  TEXT NOT NULL DEFAULT '',
    host_region        TEXT NOT NULL DEFAULT '',
    host_city          TEXT NOT NULL DEFAULT '',
    host_isp           TEXT NOT NULL DEFAULT '',
    host_org           TEXT NOT NULL DEFAULT '',
    host_as            TEXT NOT NULL DEFAULT '',
    host_lat           REAL NOT NULL DEFAULT 0,
    host_lon           REAL NOT NULL DEFAULT 0,
    -- provider detection
    dns_provider       TEXT NOT NULL DEFAULT '',
    email_provider     TEXT NOT NULL DEFAULT '',
    hosting_provider   TEXT NOT NULL DEFAULT '',
    ca_provider        TEXT NOT NULL DEFAULT '',
    headers            JSON NOT NULL DEFAULT '{}',
    favicon_url        TEXT NOT NULL DEFAULT '',
    lookup_error       TEXT NOT NULL DEFAULT '',
    -- scheduling
    interval_h         INTEGER NOT NULL DEFAULT 24,
    last_checked       TEXT,
    position           REAL NOT NULL DEFAULT 0,
    created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX domains_due ON domains(active, last_checked);

CREATE TABLE domain_checks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id    TEXT NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    expiry_date  TEXT,
    ssl_valid_to TEXT,
    ipv4         JSON NOT NULL DEFAULT '[]',
    name_servers JSON NOT NULL DEFAULT '[]',
    checked_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX domain_checks_domain_time ON domain_checks(domain_id, checked_at);

-- +goose Down
DROP TABLE domain_checks;
DROP TABLE domains;
