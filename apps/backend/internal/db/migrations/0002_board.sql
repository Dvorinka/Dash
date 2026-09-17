-- +goose Up
-- Board tree: sections > items > urls. position REAL = fractional ordering;
-- a drag writes one row instead of renumbering the set.
CREATE TABLE sections (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    position   REAL NOT NULL,
    collapsed  INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE items (
    id         TEXT PRIMARY KEY,
    section_id TEXT NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL DEFAULT 'service' CHECK (kind IN ('service', 'widget')),
    name       TEXT NOT NULL,
    icon       TEXT NOT NULL DEFAULT '',
    position   REAL NOT NULL,
    config     JSON,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX items_section_pos ON items(section_id, position);

CREATE TABLE urls (
    id       TEXT PRIMARY KEY,
    item_id  TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    url      TEXT NOT NULL,
    label    TEXT NOT NULL DEFAULT '',
    position REAL NOT NULL
);
CREATE INDEX urls_item_pos ON urls(item_id, position);

-- +goose Down
DROP TABLE urls;
DROP TABLE items;
DROP TABLE sections;
