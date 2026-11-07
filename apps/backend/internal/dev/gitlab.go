package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GitLab provider — REST API v4, works on gitlab.com and self-hosted
// (host set → {host}/api/v4). Auth: PRIVATE-TOKEN header.
type GitLab struct{}

func (GitLab) apiBase(host string) string {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		return "https://gitlab.com/api/v4"
	}
	return strings.TrimSuffix(host, "/") + "/api/v4"
}

var glHTTP = &http.Client{Timeout: 15 * time.Second}

func glDo(ctx context.Context, method, u, token string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = strings.NewReader(string(raw))
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("PRIVATE-TOKEN", token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := glHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab unreachable")
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("gitlab %d: %s", resp.StatusCode, strings.TrimSpace(string(data[:min(len(data), 200)])))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

// projectID encodes group/project for the URL path.
func projectID(path string) string {
	return url.PathEscape(strings.Trim(path, "/"))
}

func (GitLab) Project(ctx context.Context, p Project, token string) (Meta, error) {
	var r struct {
		Description   string `json:"description"`
		DefaultBranch string `json:"default_branch"`
		Statistics    struct {
			StarCount int `json:"star_count"`
			ForkCount int `json:"fork_count"`
		} `json:"statistics"`
		OpenIssuesCount int    `json:"open_issues_count"`
		Visibility      string `json:"visibility"`
		WebURL          string `json:"web_url"`
		LastActivityAt  string `json:"last_activity_at"`
		Namespace       struct {
			Kind string `json:"kind"`
		} `json:"namespace"`
	}
	var langs map[string]any
	err := glDo(ctx, http.MethodGet,
		GitLab{}.apiBase(p.Host)+"/projects/"+projectID(p.Path), token, nil, &r)
	if err == nil {
		// languages are a separate lightweight call; ignore failure
		_ = glDo(ctx, http.MethodGet,
			GitLab{}.apiBase(p.Host)+"/projects/"+projectID(p.Path)+"/languages", token, nil, &langs)
	}
	lang := ""
	if len(langs) > 0 {
		// pick the top language by byte share — approx via max value
		var best string
		var bestV float64
		for k, v := range langs {
			if f, ok := v.(float64); ok && f > bestV {
				bestV, best = f, k
			}
		}
		lang = best
	}
	return Meta{
		Description:   r.Description,
		DefaultBranch: r.DefaultBranch,
		Language:      lang,
		Stars:         r.Statistics.StarCount,
		Forks:         r.Statistics.ForkCount,
		OpenIssues:    r.OpenIssuesCount,
		Visibility:    r.Visibility,
		Homepage:      r.WebURL,
		PushedAt:      r.LastActivityAt,
	}, err
}

type glIssue struct {
	IID    int    `json:"iid"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Author struct {
		Username string `json:"username"`
	} `json:"author"`
	Labels []struct {
		Title string `json:"title"`
	} `json:"labels"`
	WebURL       string `json:"web_url"`
	Description  string `json:"description"`
	UserNotesCnt int    `json:"user_notes_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	ClosedAt     string `json:"closed_at"`
}

func (GitLab) Issues(ctx context.Context, p Project, token string) ([]Issue, error) {
	var raw []glIssue
	err := glDo(ctx, http.MethodGet,
		GitLab{}.apiBase(p.Host)+"/projects/"+projectID(p.Path)+
			"/issues?state=all&order_by=updated_at&sort=desc&per_page=100", token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]Issue, 0, len(raw))
	for _, i := range raw {
		labels := make([]string, 0, len(i.Labels))
		for _, l := range i.Labels {
			labels = append(labels, l.Title)
		}
		state := i.State
		if state == "opened" {
			state = "open"
		}
		out = append(out, Issue{
			Number: i.IID, IID: i.IID, Title: i.Title, State: state,
			Author: i.Author.Username, Labels: labels, URL: i.WebURL, Body: i.Description,
			Comments:  i.UserNotesCnt,
			CreatedAt: *ParseTime(i.CreatedAt), UpdatedAt: *ParseTime(i.UpdatedAt),
			ClosedAt: parsePtrStr(i.ClosedAt),
		})
	}
	return out, nil
}

type glMR struct {
	IID    int    `json:"iid"`
	Title  string `json:"title"`
	State  string `json:"state"` // opened | merged | closed
	Draft  bool   `json:"draft"`
	Author struct {
		Username string `json:"username"`
	} `json:"author"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	ChangesCount string `json:"changes_count"`
	UserNotesCnt int    `json:"user_notes_count"`
	WebURL       string `json:"web_url"`
	Description  string `json:"description"`
	MergeStatus  string `json:"merge_status"` // can_be_merged | cannot_be_merged | …
	DetailedMR   struct {
		Draft       bool   `json:"draft"`
		MergeStatus string `json:"merge_status"`
	} `json:"detailed_merge_status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	MergedAt  string `json:"merged_at"`
	ClosedAt  string `json:"closed_at"`
}

func glMergeable(ms string) string {
	switch ms {
	case "can_be_merged", "mergeable":
		return "mergeable"
	case "cannot_be_merged", "broken_status", "conflict", "need_rebase":
		return "conflict"
	default:
		return "unknown"
	}
}

func (GitLab) PullRequests(ctx context.Context, p Project, token string) ([]PullRequest, error) {
	var raw []glMR
	err := glDo(ctx, http.MethodGet,
		GitLab{}.apiBase(p.Host)+"/projects/"+projectID(p.Path)+
			"/merge_requests?state=all&order_by=updated_at&sort=desc&per_page=100", token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]PullRequest, 0, len(raw))
	for _, m := range raw {
		state := m.State
		if state == "opened" {
			state = "open"
		}
		ms := m.MergeStatus
		if m.DetailedMR.MergeStatus != "" {
			ms = m.DetailedMR.MergeStatus
		}
		changes, _ := strconv.Atoi(m.ChangesCount)
		out = append(out, PullRequest{
			Number: m.IID, Title: m.Title, State: state, Draft: m.Draft || m.DetailedMR.Draft,
			Author: m.Author.Username, SourceBranch: m.SourceBranch, TargetBranch: m.TargetBranch,
			ChangedFiles: changes,
			Mergeable:    glMergeable(ms),
			URL:          m.WebURL, Body: m.Description, Comments: m.UserNotesCnt,
			CreatedAt: *ParseTime(m.CreatedAt), UpdatedAt: *ParseTime(m.UpdatedAt),
			MergedAt: parsePtrStr(m.MergedAt), ClosedAt: parsePtrStr(m.ClosedAt),
		})
	}
	return out, nil
}

// Contributions has no calendar API on GitLab — count authored commits
// per day over the last year via the commits endpoint (capped pages).
func (GitLab) Contributions(ctx context.Context, p Project, token string) ([]DayCount, error) {
	since := time.Now().AddDate(-1, 0, 0).UTC().Format("2006-01-02")
	counts := map[string]int{}
	for page := 1; page <= 10; page++ {
		var raw []struct {
			AuthoredDate string `json:"authored_date"`
		}
		u := fmt.Sprintf("%s/projects/%s/repository/commits?since=%s&per_page=100&page=%d",
			GitLab{}.apiBase(p.Host), projectID(p.Path), since, page)
		if err := glDo(ctx, http.MethodGet, u, token, nil, &raw); err != nil {
			return nil, err
		}
		for _, c := range raw {
			day := c.AuthoredDate
			if len(day) > 10 {
				day = day[:10]
			}
			counts[day]++
		}
		if len(raw) < 100 {
			break
		}
	}
	out := make([]DayCount, 0, len(counts))
	for d, n := range counts {
		out = append(out, DayCount{Date: d, Count: n})
	}
	return out, nil
}

func (GitLab) MergePR(ctx context.Context, p Project, token string, number int, method string) error {
	body := map[string]any{}
	if method == "squash" {
		body["squash"] = true
	}
	return glDo(ctx, http.MethodPut,
		fmt.Sprintf("%s/projects/%s/merge_requests/%d/merge", GitLab{}.apiBase(p.Host), projectID(p.Path), number),
		token, body, nil)
}

func (GitLab) ClosePR(ctx context.Context, p Project, token string, number int) error {
	return glDo(ctx, http.MethodPut,
		fmt.Sprintf("%s/projects/%s/merge_requests/%d", GitLab{}.apiBase(p.Host), projectID(p.Path), number),
		token, map[string]any{"state_event": "close"}, nil)
}

func (GitLab) CloseIssue(ctx context.Context, p Project, token string, number int) error {
	return glDo(ctx, http.MethodPut,
		fmt.Sprintf("%s/projects/%s/issues/%d", GitLab{}.apiBase(p.Host), projectID(p.Path), number),
		token, map[string]any{"state_event": "close"}, nil)
}

func (GitLab) ReopenPR(ctx context.Context, p Project, token string, number int) error {
	return glDo(ctx, http.MethodPut,
		fmt.Sprintf("%s/projects/%s/merge_requests/%d", GitLab{}.apiBase(p.Host), projectID(p.Path), number),
		token, map[string]any{"state_event": "reopen"}, nil)
}

func (GitLab) ReopenIssue(ctx context.Context, p Project, token string, number int) error {
	return glDo(ctx, http.MethodPut,
		fmt.Sprintf("%s/projects/%s/issues/%d", GitLab{}.apiBase(p.Host), projectID(p.Path), number),
		token, map[string]any{"state_event": "reopen"}, nil)
}

func (GitLab) CommentIssue(ctx context.Context, p Project, token string, number int, body string) error {
	return glDo(ctx, http.MethodPost,
		fmt.Sprintf("%s/projects/%s/issues/%d/notes", GitLab{}.apiBase(p.Host), projectID(p.Path), number),
		token, map[string]any{"body": body}, nil)
}

func (GitLab) CommentPR(ctx context.Context, p Project, token string, number int, body string) error {
	return glDo(ctx, http.MethodPost,
		fmt.Sprintf("%s/projects/%s/merge_requests/%d/notes", GitLab{}.apiBase(p.Host), projectID(p.Path), number),
		token, map[string]any{"body": body}, nil)
}

func (GitLab) Comments(ctx context.Context, p Project, token string, kind string, number int) ([]ItemComment, error) {
	coll := "issues"
	if kind == "pr" {
		coll = "merge_requests"
	}
	var raw []struct {
		Author struct {
			Username string `json:"username"`
		} `json:"author"`
		Body      string `json:"body"`
		System    bool   `json:"system"`
		CreatedAt string `json:"created_at"`
	}
	err := glDo(ctx, http.MethodGet,
		fmt.Sprintf("%s/projects/%s/%s/%d/notes?per_page=100&sort=asc",
			GitLab{}.apiBase(p.Host), projectID(p.Path), coll, number), token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]ItemComment, 0, len(raw))
	for _, n := range raw {
		if n.System {
			continue
		}
		out = append(out, ItemComment{
			Author: n.Author.Username, Body: n.Body,
			CreatedAt: *ParseTime(n.CreatedAt),
		})
	}
	return out, nil
}

func (GitLab) PRFiles(ctx context.Context, p Project, token string, number int) ([]DiffFile, error) {
	var raw []struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
		Diff    string `json:"diff"`
		NewFile bool   `json:"new_file"`
		Renamed bool   `json:"renamed_file"`
		Deleted bool   `json:"deleted_file"`
	}
	err := glDo(ctx, http.MethodGet,
		fmt.Sprintf("%s/projects/%s/merge_requests/%d/diffs?per_page=100",
			GitLab{}.apiBase(p.Host), projectID(p.Path), number), token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]DiffFile, 0, len(raw))
	for _, f := range raw {
		status := "modified"
		switch {
		case f.NewFile:
			status = "added"
		case f.Deleted:
			status = "removed"
		case f.Renamed:
			status = "renamed"
		}
		adds, dels := 0, 0
		for _, line := range strings.Split(f.Diff, "\n") {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				adds++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				dels++
			}
		}
		out = append(out, DiffFile{
			Filename: f.NewPath, Status: status,
			Additions: adds, Deletions: dels, Patch: f.Diff,
		})
	}
	return out, nil
}

func (GitLab) Discover(ctx context.Context, host, token string) ([]RepoRef, error) {
	var raw []struct {
		PathWithNamespace string `json:"path_with_namespace"`
		Name              string `json:"name"`
		Visibility        string `json:"visibility"`
	}
	err := glDo(ctx, http.MethodGet,
		GitLab{}.apiBase(host)+"/projects?membership=true&per_page=100&simple=true&order_by=last_activity_at",
		token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]RepoRef, 0, len(raw))
	for _, r := range raw {
		out = append(out, RepoRef{
			Path: r.PathWithNamespace, Name: r.Name, Host: host,
			Private: r.Visibility != "public",
		})
	}
	return out, nil
}

func parsePtrStr(s string) *time.Time {
	if s == "" {
		return nil
	}
	return ParseTime(s)
}
