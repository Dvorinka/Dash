// Package dev integrates Dash with git hosting providers — GitHub and
// GitLab — syncing projects, issues, pull requests and contribution
// activity into SQLite. Provider logic is pure HTTP; storage lives in
// internal/api.
package dev

import (
	"context"
	"fmt"
	"time"
)

// Project is one tracked repository (or GitLab project).
type Project struct {
	ID             string     `json:"id"`
	Provider       string     `json:"provider"` // github | gitlab
	Host           string     `json:"host"`
	Path           string     `json:"path"`
	Name           string     `json:"name"`
	Token          string     `json:"-"` // never serialized
	InstallationID string     `json:"-"`
	Enabled        bool       `json:"enabled"`
	Position       float64    `json:"position"`
	LastSyncAt     *time.Time `json:"lastSyncAt"`
	LastSyncError  string     `json:"lastSyncError"`
	Meta           Meta       `json:"meta"`
	OpenIssues     int        `json:"openIssues"`
	OpenPRs        int        `json:"openPRs"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// Meta is the provider's repo summary (stars, forks, language…).
type Meta struct {
	Description   string `json:"description"`
	DefaultBranch string `json:"defaultBranch"`
	Language      string `json:"language"`
	Stars         int    `json:"stars"`
	Forks         int    `json:"forks"`
	OpenIssues    int    `json:"openIssues"`
	Visibility    string `json:"visibility"`
	Homepage      string `json:"homepage"`
	PushedAt      string `json:"pushedAt"`
}

// Issue is a provider issue (GitHub issues / GitLab issues).
type Issue struct {
	Number    int        `json:"number"`
	IID       int        `json:"iid"`
	Title     string     `json:"title"`
	State     string     `json:"state"` // open | closed
	Author    string     `json:"author"`
	Labels    []string   `json:"labels"`
	URL       string     `json:"url"`
	Body      string     `json:"body"`
	Comments  int        `json:"comments"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	ClosedAt  *time.Time `json:"closedAt"`
}

// PullRequest is a provider merge/pull request.
type PullRequest struct {
	Number         int        `json:"number"`
	Title          string     `json:"title"`
	State          string     `json:"state"` // open | merged | closed
	Draft          bool       `json:"draft"`
	Author         string     `json:"author"`
	SourceBranch   string     `json:"sourceBranch"`
	TargetBranch   string     `json:"targetBranch"`
	Additions      int        `json:"additions"`
	Deletions      int        `json:"deletions"`
	ChangedFiles   int        `json:"changedFiles"`
	Mergeable      string     `json:"mergeable"`      // mergeable | conflict | unknown
	ReviewDecision string     `json:"reviewDecision"` // approved | changes_requested | review_required | ""
	URL            string     `json:"url"`
	Body           string     `json:"body"`
	Comments       int        `json:"comments"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	MergedAt       *time.Time `json:"mergedAt"`
	ClosedAt       *time.Time `json:"closedAt"`
}

// DayCount is one cell of the contribution heatmap.
type DayCount struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Count int    `json:"count"`
}

// RepoRef is a discoverable repository returned by Discover.
type RepoRef struct {
	Path    string `json:"path"` // owner/repo or group/project
	Name    string `json:"name"`
	Host    string `json:"host"` // empty = provider default
	Private bool   `json:"private"`
}

// Provider is one git host implementation.
type Provider interface {
	// Project fetches repo metadata.
	Project(ctx context.Context, p Project, token string) (Meta, error)
	// Issues returns recent issues (open + recently closed).
	Issues(ctx context.Context, p Project, token string) ([]Issue, error)
	// PullRequests returns recent PRs/MRs.
	PullRequests(ctx context.Context, p Project, token string) ([]PullRequest, error)
	// Contributions returns per-day activity for the heatmap.
	Contributions(ctx context.Context, p Project, token string) ([]DayCount, error)
	// MergePR merges an open PR/MR. method: merge | squash | rebase.
	MergePR(ctx context.Context, p Project, token string, number int, method string) error
	// ClosePR closes an open PR/MR without merging.
	ClosePR(ctx context.Context, p Project, token string, number int) error
	// CloseIssue closes an issue.
	CloseIssue(ctx context.Context, p Project, token string, number int) error
	// Discover lists repos the token can see (auto-detect).
	Discover(ctx context.Context, host, token string) ([]RepoRef, error)
}

// For returns the provider for a project row.
func For(provider string) (Provider, error) {
	switch provider {
	case "github":
		return GitHub{}, nil
	case "gitlab":
		return GitLab{}, nil
	default:
		return nil, fmt.Errorf("unknown provider %q", provider)
	}
}

// ParseTime tolerates the providers' assorted timestamp formats.
func ParseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, f := range []string{time.RFC3339, "2006-01-02T15:04:05.999Z", "2006-01-02 15:04:05 -0700 MST"} {
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	return nil
}
