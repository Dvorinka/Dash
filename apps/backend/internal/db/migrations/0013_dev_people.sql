-- +goose Up
-- Assignees on issues/PRs and requested reviewers on PRs, synced from
-- providers and managed through the dev API.
ALTER TABLE dev_issues ADD COLUMN assignees TEXT NOT NULL DEFAULT '[]';
ALTER TABLE dev_prs ADD COLUMN assignees TEXT NOT NULL DEFAULT '[]';
ALTER TABLE dev_prs ADD COLUMN reviewers TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE dev_prs DROP COLUMN reviewers;
ALTER TABLE dev_prs DROP COLUMN assignees;
ALTER TABLE dev_issues DROP COLUMN assignees;
