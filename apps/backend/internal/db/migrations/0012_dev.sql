-- +goose Up
-- Developer integrations: git hosting projects (GitHub/GitLab), their issues
-- and pull requests, synced on a schedule. Tokens live per-project; global
-- fallbacks live in the settings table (dev.githubToken etc.).
CREATE TABLE dev_projects (
    id              TEXT PRIMARY KEY,
    provider        TEXT NOT NULL,             -- github | gitlab
    host            TEXT NOT NULL DEFAULT '',  -- empty = provider default
    path            TEXT NOT NULL,             -- owner/repo or group/project
    name            TEXT NOT NULL DEFAULT '',
    token           TEXT NOT NULL DEFAULT '',  -- PAT; empty -> settings/app
    installation_id TEXT NOT NULL DEFAULT '',  -- GitHub App installation
    enabled         INTEGER NOT NULL DEFAULT 1,
    position        REAL NOT NULL DEFAULT 1024,
    last_sync_at    TEXT,
    last_sync_error TEXT NOT NULL DEFAULT '',
    meta            TEXT NOT NULL DEFAULT '{}',
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE dev_issues (
    project_id   TEXT NOT NULL,
    number       INTEGER NOT NULL,
    iid          INTEGER NOT NULL DEFAULT 0,
    title        TEXT NOT NULL DEFAULT '',
    state        TEXT NOT NULL DEFAULT 'open',
    author       TEXT NOT NULL DEFAULT '',
    labels       TEXT NOT NULL DEFAULT '[]',
    url          TEXT NOT NULL DEFAULT '',
    body         TEXT NOT NULL DEFAULT '',
    comments     INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT,
    updated_at   TEXT,
    closed_at    TEXT,
    PRIMARY KEY (project_id, number)
);
CREATE INDEX idx_dev_issues_state ON dev_issues(state);

CREATE TABLE dev_prs (
    project_id      TEXT NOT NULL,
    number          INTEGER NOT NULL,
    title           TEXT NOT NULL DEFAULT '',
    state           TEXT NOT NULL DEFAULT 'open',
    draft           INTEGER NOT NULL DEFAULT 0,
    author          TEXT NOT NULL DEFAULT '',
    source_branch   TEXT NOT NULL DEFAULT '',
    target_branch   TEXT NOT NULL DEFAULT '',
    additions       INTEGER NOT NULL DEFAULT 0,
    deletions       INTEGER NOT NULL DEFAULT 0,
    changed_files   INTEGER NOT NULL DEFAULT 0,
    mergeable       TEXT NOT NULL DEFAULT '',
    review_decision TEXT NOT NULL DEFAULT '',
    url             TEXT NOT NULL DEFAULT '',
    body            TEXT NOT NULL DEFAULT '',
    comments        INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT,
    updated_at      TEXT,
    merged_at       TEXT,
    closed_at       TEXT,
    PRIMARY KEY (project_id, number)
);
CREATE INDEX idx_dev_prs_state ON dev_prs(state);

-- +goose Down
DROP TABLE dev_prs;
DROP TABLE dev_issues;
DROP TABLE dev_projects;
