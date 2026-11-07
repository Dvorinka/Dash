-- +goose Up
-- Labels on PRs/MRs (issues already carry them since 0012).
ALTER TABLE dev_prs ADD COLUMN labels TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE dev_prs DROP COLUMN labels;
