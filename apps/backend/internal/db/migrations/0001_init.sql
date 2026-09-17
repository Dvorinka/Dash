-- +goose Up
-- Key/value store for user prefs (theme, renderer, ...). Values are JSON.
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value JSON NOT NULL
);

-- +goose Down
DROP TABLE settings;
