-- +goose Up
-- Per-target alert rules as a JSON blob: {consecutiveFailures, latencyWarnMs,
-- certDays, mute}. '{}' = defaults (alert on first failure, no latency rule,
-- cert threshold = alert_days_before, unmuted).
ALTER TABLE monitors ADD COLUMN alerts JSON NOT NULL DEFAULT '{}';
ALTER TABLE domains  ADD COLUMN alerts JSON NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE monitors DROP COLUMN alerts;
ALTER TABLE domains  DROP COLUMN alerts;
