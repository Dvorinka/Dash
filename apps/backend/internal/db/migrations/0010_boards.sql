-- +goose Up
-- Multiple boards. Every section belongs to exactly one board; the seeded
-- 'home' board adopts existing rows. NULL board_id maps to 'home' at read
-- time so old imports/exports keep working without a backfill.
CREATE TABLE boards (
    id         TEXT PRIMARY KEY,
    slug       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    position   REAL NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
INSERT INTO boards (id, slug, name, position) VALUES ('b_home', 'home', 'Home', 1024);

ALTER TABLE sections ADD COLUMN board_id TEXT REFERENCES boards(id) ON DELETE CASCADE;
UPDATE sections SET board_id = 'b_home';
CREATE INDEX sections_board_pos ON sections(board_id, position);

-- +goose Down
DROP INDEX IF EXISTS sections_board_pos;
-- SQLite can't drop columns pre-3.35; rebuild instead.
CREATE TABLE sections_new (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    position   REAL NOT NULL,
    collapsed  INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
INSERT INTO sections_new (id, name, position, collapsed, created_at)
    SELECT id, name, position, collapsed, created_at FROM sections;
DROP TABLE sections;
ALTER TABLE sections_new RENAME TO sections;
DROP TABLE boards;
