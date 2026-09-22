-- +goose Up
-- Structured DNS records + provider identifications for the domain detail
-- page. JSON blobs — the flat columns stay for compat and check history.
ALTER TABLE domains ADD COLUMN records   TEXT NOT NULL DEFAULT '[]';
ALTER TABLE domains ADD COLUMN providers TEXT NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE domains DROP COLUMN providers;
ALTER TABLE domains DROP COLUMN records;
