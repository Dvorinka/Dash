package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Gitea provider — REST API v1. Also serves Forgejo ("forgejo" aliases
// here) since the API surface is compatible. Auth: Authorization: token.
// Default host is gitea.com; self-hosted instances set Project.Host.
type Gitea struct{}

func (Gitea) apiBase(host string) string {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		return "https://gitea.com/api/v1"
	}
	return strings.TrimSuffix(host, "/") + "/api/v1"
}

var gtHTTP = &http.Client{Timeout: 15 * time.Second}

func gtDo(ctx context.Context, method, u, token string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := gtHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("gitea unreachable")
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("gitea %d: %s", resp.StatusCode, strings.TrimSpace(string(data[:min(len(data), 200)])))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func (Gitea) Project(ctx context.Context, p Project, token string) (Meta, error) {
	var r struct {
		Description   string `json:"description"`
		DefaultBranch string `json:"default_branch"`
		Language      string `json:"language"`
		Stars         int    `json:"stargazers_count"`
		Forks         int    `json:"forks_count"`
		OpenIssues    int    `json:"open_issues_count"`
		Visibility    string `json:"visibility"`
		Homepage      string `json:"homepage"`
		UpdatedAt     string `json:"updated_at"`
	}
	err := gtDo(ctx, http.MethodGet,
		Gitea{}.apiBase(p.Host)+"/repos/"+p.Path, token, nil, &r)
	return Meta{
		Description:   r.Description,
		DefaultBranch: r.DefaultBranch,
		Language:      r.Language,
		Stars:         r.Stars,
		Forks:         r.Forks,
		OpenIssues:    r.OpenIssues,
		Visibility:    r.Visibility,
		Homepage:      r.Homepage,
		PushedAt:      r.UpdatedAt,
	}, err
}

type gtIssue struct {
	Index int    `json:"index"`
	Title string `json:"title"`
	State string `json:"state"`
	User  struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	HTMLURL   string `json:"html_url"`
	Body      string `json:"body"`
	Comments  int    `json:"comments"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	// pull_request is non-null on PRs leaked into /issues.
	PullRequest *struct{} `json:"pull_request"`
}

func (Gitea) Issues(ctx context.Context, p Project, token string) ([]Issue, error) {
	var raw []gtIssue
	err := gtDo(ctx, http.MethodGet,
		Gitea{}.apiBase(p.Host)+"/repos/"+p.Path+
			"/issues?state=all&type=issues&limit=50&sort=recentupdate", token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]Issue, 0, len(raw))
	for _, i := range raw {
		if i.PullRequest != nil {
			continue
		}
		labels := make([]string, 0, len(i.Labels))
		for _, l := range i.Labels {
			labels = append(labels, l.Name)
		}
		out = append(out, Issue{
			Number: i.Index, IID: i.Index, Title: i.Title, State: i.State,
			Author: i.User.Login, Labels: labels, URL: i.HTMLURL, Body: i.Body,
			Comments:  i.Comments,
			CreatedAt: *ParseTime(i.CreatedAt), UpdatedAt: *ParseTime(i.UpdatedAt),
		})
	}
	return out, nil
}

type gtPR struct {
	Index  int    `json:"index"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Draft  bool   `json:"draft"`
	Merged bool   `json:"merged"`
	User   struct {
		Login string `json:"login"`
	} `json:"user"`
	Head struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	ChangedFiles int    `json:"changed_files"`
	Mergeable    *bool  `json:"mergeable"`
	HTMLURL      string `json:"html_url"`
	Body         string `json:"body"`
	Comments     int    `json:"comments"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	MergedAt     string `json:"merged_at"`
	ClosedAt     string `json:"closed_at"`
}

func (Gitea) PullRequests(ctx context.Context, p Project, token string) ([]PullRequest, error) {
	var raw []gtPR
	err := gtDo(ctx, http.MethodGet,
		Gitea{}.apiBase(p.Host)+"/repos/"+p.Path+
			"/pulls?state=all&limit=50&sort=recentupdate", token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]PullRequest, 0, len(raw))
	for _, pr := range raw {
		state := pr.State
		if pr.Merged {
			state = "merged"
		}
		mergeable := "unknown"
		if pr.Mergeable != nil {
			if *pr.Mergeable {
				mergeable = "mergeable"
			} else {
				mergeable = "conflict"
			}
		}
		out = append(out, PullRequest{
			Number: pr.Index, Title: pr.Title, State: state, Draft: pr.Draft,
			Author: pr.User.Login, SourceBranch: pr.Head.Ref, TargetBranch: pr.Base.Ref,
			Additions: pr.Additions, Deletions: pr.Deletions, ChangedFiles: pr.ChangedFiles,
			Mergeable: mergeable,
			URL:       pr.HTMLURL, Body: pr.Body, Comments: pr.Comments,
			CreatedAt: *ParseTime(pr.CreatedAt), UpdatedAt: *ParseTime(pr.UpdatedAt),
			MergedAt: parsePtrStr(pr.MergedAt), ClosedAt: parsePtrStr(pr.ClosedAt),
		})
	}
	return out, nil
}

// Contributions counts authored commits per day — Gitea has no calendar
// endpoint, so this mirrors the GitLab approach.
func (Gitea) Contributions(ctx context.Context, p Project, token string) ([]DayCount, error) {
	since := time.Now().AddDate(-1, 0, 0).UTC().Format("2006-01-02")
	counts := map[string]int{}
	for page := 1; page <= 10; page++ {
		var raw []struct {
			Commit struct {
				Author struct {
					Date string `json:"date"`
				} `json:"author"`
			} `json:"commit"`
		}
		u := fmt.Sprintf("%s/repos/%s/commits?since=%s&limit=50&page=%d",
			Gitea{}.apiBase(p.Host), p.Path, since, page)
		if err := gtDo(ctx, http.MethodGet, u, token, nil, &raw); err != nil {
			return nil, err
		}
		for _, c := range raw {
			day := c.Commit.Author.Date
			if len(day) > 10 {
				day = day[:10]
			}
			if day != "" {
				counts[day]++
			}
		}
		if len(raw) < 50 {
			break
		}
	}
	out := make([]DayCount, 0, len(counts))
	for d, n := range counts {
		out = append(out, DayCount{Date: d, Count: n})
	}
	return out, nil
}

func (Gitea) MergePR(ctx context.Context, p Project, token string, number int, method string) error {
	do := "merge"
	switch method {
	case "squash", "rebase":
		do = method
	}
	return gtDo(ctx, http.MethodPost,
		fmt.Sprintf("%s/repos/%s/pulls/%d/merge", Gitea{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"Do": do}, nil)
}

func (Gitea) ClosePR(ctx context.Context, p Project, token string, number int) error {
	return gtDo(ctx, http.MethodPatch,
		fmt.Sprintf("%s/repos/%s/pulls/%d", Gitea{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"state": "closed"}, nil)
}

func (Gitea) ReopenPR(ctx context.Context, p Project, token string, number int) error {
	return gtDo(ctx, http.MethodPatch,
		fmt.Sprintf("%s/repos/%s/pulls/%d", Gitea{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"state": "open"}, nil)
}

func (Gitea) CloseIssue(ctx context.Context, p Project, token string, number int) error {
	return gtDo(ctx, http.MethodPatch,
		fmt.Sprintf("%s/repos/%s/issues/%d", Gitea{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"state": "closed"}, nil)
}

func (Gitea) ReopenIssue(ctx context.Context, p Project, token string, number int) error {
	return gtDo(ctx, http.MethodPatch,
		fmt.Sprintf("%s/repos/%s/issues/%d", Gitea{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"state": "open"}, nil)
}

// CommentIssue and CommentPR share Gitea's issue-comments endpoint —
// pull request conversations are issue comments.
func (Gitea) CommentIssue(ctx context.Context, p Project, token string, number int, body string) error {
	return gtDo(ctx, http.MethodPost,
		fmt.Sprintf("%s/repos/%s/issues/%d/comments", Gitea{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"body": body}, nil)
}

func (Gitea) CommentPR(ctx context.Context, p Project, token string, number int, body string) error {
	return Gitea{}.CommentIssue(ctx, p, token, number, body)
}

func (Gitea) Comments(ctx context.Context, p Project, token string, kind string, number int) ([]ItemComment, error) {
	var raw []struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		Body      string `json:"body"`
		CreatedAt string `json:"created_at"`
		HTMLURL   string `json:"html_url"`
	}
	err := gtDo(ctx, http.MethodGet,
		fmt.Sprintf("%s/repos/%s/issues/%d/comments",
			Gitea{}.apiBase(p.Host), p.Path, number), token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]ItemComment, 0, len(raw))
	for _, c := range raw {
		out = append(out, ItemComment{
			Author: c.User.Login, Body: c.Body, URL: c.HTMLURL,
			CreatedAt: *ParseTime(c.CreatedAt),
		})
	}
	return out, nil
}

func (Gitea) PRFiles(ctx context.Context, p Project, token string, number int) ([]DiffFile, error) {
	var raw []struct {
		Filename  string `json:"filename"`
		Status    string `json:"status"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
		Patch     string `json:"patch"`
	}
	err := gtDo(ctx, http.MethodGet,
		fmt.Sprintf("%s/repos/%s/pulls/%d/files",
			Gitea{}.apiBase(p.Host), p.Path, number), token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]DiffFile, 0, len(raw))
	for _, f := range raw {
		out = append(out, DiffFile{
			Filename: f.Filename, Status: f.Status,
			Additions: f.Additions, Deletions: f.Deletions, Patch: f.Patch,
		})
	}
	return out, nil
}

func (Gitea) Discover(ctx context.Context, host, token string) ([]RepoRef, error) {
	var raw []struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
		Private  bool   `json:"private"`
	}
	err := gtDo(ctx, http.MethodGet,
		Gitea{}.apiBase(host)+"/user/repos?limit=50", token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]RepoRef, 0, len(raw))
	for _, r := range raw {
		out = append(out, RepoRef{Path: r.FullName, Name: r.Name, Host: host, Private: r.Private})
	}
	return out, nil
}
