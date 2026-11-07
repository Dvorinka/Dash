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

// GitHub provider — REST for repo data, GraphQL for the contribution
// calendar (REST has no equivalent). Works against github.com and GHES
// (host set → {host}/api/v3 + {host}/api/graphql).
type GitHub struct{}

func (GitHub) apiBase(host string) string {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		return "https://api.github.com"
	}
	return strings.TrimSuffix(host, "/") + "/api/v3"
}

func (GitHub) gqlURL(host string) string {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		return "https://api.github.com/graphql"
	}
	return strings.TrimSuffix(host, "/") + "/api/graphql"
}

var ghHTTP = &http.Client{Timeout: 15 * time.Second}

func ghDo(ctx context.Context, method, u, token string, body any, out any) error {
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
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := ghHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("github unreachable")
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("github %d: %s", resp.StatusCode, strings.TrimSpace(string(data[:min(len(data), 200)])))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func (GitHub) Project(ctx context.Context, p Project, token string) (Meta, error) {
	var r struct {
		Description   string `json:"description"`
		DefaultBranch string `json:"default_branch"`
		Language      string `json:"language"`
		Stargazers    int    `json:"stargazers_count"`
		Forks         int    `json:"forks_count"`
		OpenIssues    int    `json:"open_issues_count"`
		Visibility    string `json:"visibility"`
		Homepage      string `json:"homepage"`
		PushedAt      string `json:"pushed_at"`
	}
	err := ghDo(ctx, http.MethodGet,
		GitHub{}.apiBase(p.Host)+"/repos/"+p.Path, token, nil, &r)
	return Meta{
		Description:   r.Description,
		DefaultBranch: r.DefaultBranch,
		Language:      r.Language,
		Stars:         r.Stargazers,
		Forks:         r.Forks,
		OpenIssues:    r.OpenIssues,
		Visibility:    r.Visibility,
		Homepage:      r.Homepage,
		PushedAt:      r.PushedAt,
	}, err
}

type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	User   struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Assignees []struct {
		Login string `json:"login"`
	} `json:"assignees"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
	Comments    int       `json:"comments"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
	ClosedAt    *string   `json:"closed_at"`
	PullRequest *struct{} `json:"pull_request"` // PRs leak into /issues
}

func (GitHub) Issues(ctx context.Context, p Project, token string) ([]Issue, error) {
	var raw []ghIssue
	err := ghDo(ctx, http.MethodGet,
		GitHub{}.apiBase(p.Host)+"/repos/"+p.Path+
			"/issues?state=all&sort=updated&direction=desc&per_page=100", token, nil, &raw)
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
		assignees := make([]string, 0, len(i.Assignees))
		for _, a := range i.Assignees {
			assignees = append(assignees, a.Login)
		}
		out = append(out, Issue{
			Number: i.Number, Title: i.Title, State: i.State,
			Author: i.User.Login, Assignees: assignees, Labels: labels,
			URL: i.HTMLURL, Body: i.Body,
			Comments:  i.Comments,
			CreatedAt: *ParseTime(i.CreatedAt), UpdatedAt: *ParseTime(i.UpdatedAt),
			ClosedAt: parsePtr(i.ClosedAt),
		})
	}
	return out, nil
}

type ghPR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Draft  bool   `json:"draft"`
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
	MergeableSt  string `json:"mergeable_state"`
	HTMLURL      string `json:"html_url"`
	Body         string `json:"body"`
	Comments     int    `json:"comments"`
	Assignees    []struct {
		Login string `json:"login"`
	} `json:"assignees"`
	RequestedReviewers []struct {
		Login string `json:"login"`
	} `json:"requested_reviewers"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	MergedAt  *string `json:"merged_at"`
	ClosedAt  *string `json:"closed_at"`
}

func prMergeable(m *bool, st string) string {
	if st == "dirty" || st == "behind" {
		return "conflict"
	}
	if m != nil {
		if *m {
			return "mergeable"
		}
		return "conflict"
	}
	return "unknown"
}

func (GitHub) PullRequests(ctx context.Context, p Project, token string) ([]PullRequest, error) {
	var raw []ghPR
	err := ghDo(ctx, http.MethodGet,
		GitHub{}.apiBase(p.Host)+"/repos/"+p.Path+
			"/pulls?state=all&sort=updated&direction=desc&per_page=100", token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]PullRequest, 0, len(raw))
	for _, pr := range raw {
		state := pr.State
		if pr.MergedAt != nil {
			state = "merged"
		}
		assignees := make([]string, 0, len(pr.Assignees))
		for _, a := range pr.Assignees {
			assignees = append(assignees, a.Login)
		}
		reviewers := make([]string, 0, len(pr.RequestedReviewers))
		for _, r := range pr.RequestedReviewers {
			reviewers = append(reviewers, r.Login)
		}
		out = append(out, PullRequest{
			Number: pr.Number, Title: pr.Title, State: state, Draft: pr.Draft,
			Author: pr.User.Login, Assignees: assignees, Reviewers: reviewers,
			SourceBranch: pr.Head.Ref, TargetBranch: pr.Base.Ref,
			Additions: pr.Additions, Deletions: pr.Deletions, ChangedFiles: pr.ChangedFiles,
			Mergeable: prMergeable(pr.Mergeable, pr.MergeableSt),
			URL:       pr.HTMLURL, Body: pr.Body, Comments: pr.Comments,
			CreatedAt: *ParseTime(pr.CreatedAt), UpdatedAt: *ParseTime(pr.UpdatedAt),
			MergedAt: parsePtr(pr.MergedAt), ClosedAt: parsePtr(pr.ClosedAt),
		})
	}
	// The list endpoint omits diff stats and mergeable; enrich open PRs
	// (bounded) with a per-PR fetch.
	enriched := 0
	for i := range out {
		if out[i].State != "open" || enriched >= 25 {
			continue
		}
		enriched++
		var detail ghPR
		if err := ghDo(ctx, http.MethodGet,
			fmt.Sprintf("%s/repos/%s/pulls/%d", GitHub{}.apiBase(p.Host), p.Path, out[i].Number),
			token, nil, &detail); err == nil {
			out[i].Additions, out[i].Deletions, out[i].ChangedFiles =
				detail.Additions, detail.Deletions, detail.ChangedFiles
			out[i].Mergeable = prMergeable(detail.Mergeable, detail.MergeableSt)
		}
	}
	return out, nil
}

// Contributions uses GraphQL contributionsCollection — the only API that
// exposes the calendar. Owner is derived from the project path.
func (GitHub) Contributions(ctx context.Context, p Project, token string) ([]DayCount, error) {
	owner, _, ok := strings.Cut(p.Path, "/")
	if !ok {
		return nil, fmt.Errorf("invalid repo path")
	}
	query := `query($login:String!,$from:DateTime!,$to:DateTime!){
		user(login:$login){ contributionsCollection(from:$from,to:$to){
			contributionCalendar{ totalContributions weeks{ contributionDays{ contributionCount date } } }
		}}}`
	from := time.Now().AddDate(-1, 0, 0).UTC().Format(time.RFC3339)
	to := time.Now().UTC().Format(time.RFC3339)
	var resp struct {
		Data struct {
			User *struct {
				ContributionsCollection struct {
					ContributionCalendar struct {
						Total int `json:"totalContributions"`
						Weeks []struct {
							Days []struct {
								Count int    `json:"contributionCount"`
								Date  string `json:"date"`
							} `json:"contributionDays"`
						} `json:"weeks"`
					} `json:"contributionCalendar"`
				} `json:"contributionsCollection"`
			} `json:"user"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	body := map[string]any{"query": query, "variables": map[string]any{
		"login": owner, "from": from, "to": to,
	}}
	if err := ghDo(ctx, http.MethodPost, GitHub{}.gqlURL(p.Host), token, body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("github graphql: %s", resp.Errors[0].Message)
	}
	if resp.Data.User == nil {
		return nil, fmt.Errorf("github user %s not found", owner)
	}
	var out []DayCount
	for _, w := range resp.Data.User.ContributionsCollection.ContributionCalendar.Weeks {
		for _, d := range w.Days {
			out = append(out, DayCount{Date: d.Date, Count: d.Count})
		}
	}
	return out, nil
}

func (GitHub) MergePR(ctx context.Context, p Project, token string, number int, method string) error {
	switch method {
	case "squash", "rebase", "merge":
	default:
		method = "merge"
	}
	return ghDo(ctx, http.MethodPut,
		fmt.Sprintf("%s/repos/%s/pulls/%d/merge", GitHub{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"merge_method": method}, nil)
}

func (GitHub) ClosePR(ctx context.Context, p Project, token string, number int) error {
	return ghDo(ctx, http.MethodPatch,
		fmt.Sprintf("%s/repos/%s/pulls/%d", GitHub{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"state": "closed"}, nil)
}

func (GitHub) CloseIssue(ctx context.Context, p Project, token string, number int) error {
	return ghDo(ctx, http.MethodPatch,
		fmt.Sprintf("%s/repos/%s/issues/%d", GitHub{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"state": "closed"}, nil)
}

func (GitHub) ReopenPR(ctx context.Context, p Project, token string, number int) error {
	return ghDo(ctx, http.MethodPatch,
		fmt.Sprintf("%s/repos/%s/pulls/%d", GitHub{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"state": "open"}, nil)
}

func (GitHub) ReopenIssue(ctx context.Context, p Project, token string, number int) error {
	return ghDo(ctx, http.MethodPatch,
		fmt.Sprintf("%s/repos/%s/issues/%d", GitHub{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"state": "open"}, nil)
}

// CommentIssue and CommentPR share GitHub's issue-comments endpoint —
// pull requests are issues underneath.
func (GitHub) CommentIssue(ctx context.Context, p Project, token string, number int, body string) error {
	return ghDo(ctx, http.MethodPost,
		fmt.Sprintf("%s/repos/%s/issues/%d/comments", GitHub{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"body": body}, nil)
}

func (GitHub) CommentPR(ctx context.Context, p Project, token string, number int, body string) error {
	return GitHub{}.CommentIssue(ctx, p, token, number, body)
}

// Comments lists issue/PR conversation comments — GitHub serves both
// from the issues comments endpoint.
func (GitHub) Comments(ctx context.Context, p Project, token string, kind string, number int) ([]ItemComment, error) {
	var raw []struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		Body      string `json:"body"`
		CreatedAt string `json:"created_at"`
		HTMLURL   string `json:"html_url"`
	}
	err := ghDo(ctx, http.MethodGet,
		fmt.Sprintf("%s/repos/%s/issues/%d/comments?per_page=100",
			GitHub{}.apiBase(p.Host), p.Path, number), token, nil, &raw)
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

func (GitHub) PRFiles(ctx context.Context, p Project, token string, number int) ([]DiffFile, error) {
	var raw []struct {
		Filename  string `json:"filename"`
		Status    string `json:"status"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
		Patch     string `json:"patch"`
	}
	err := ghDo(ctx, http.MethodGet,
		fmt.Sprintf("%s/repos/%s/pulls/%d/files?per_page=100",
			GitHub{}.apiBase(p.Host), p.Path, number), token, nil, &raw)
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

// SetAssignee uses the issues assignees endpoint — pull requests share it.
func (GitHub) SetAssignee(ctx context.Context, p Project, token string, kind string, number int, username string, assign bool) error {
	method := http.MethodPost
	if !assign {
		method = http.MethodDelete
	}
	return ghDo(ctx, method,
		fmt.Sprintf("%s/repos/%s/issues/%d/assignees", GitHub{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"assignees": []string{username}}, nil)
}

func (GitHub) SetReviewer(ctx context.Context, p Project, token string, number int, username string, request bool) error {
	method := http.MethodPost
	if !request {
		method = http.MethodDelete
	}
	return ghDo(ctx, method,
		fmt.Sprintf("%s/repos/%s/pulls/%d/requested_reviewers", GitHub{}.apiBase(p.Host), p.Path, number),
		token, map[string]any{"reviewers": []string{username}}, nil)
}

func (GitHub) Discover(ctx context.Context, host, token string) ([]RepoRef, error) {
	var raw []struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
		Private  bool   `json:"private"`
	}
	err := ghDo(ctx, http.MethodGet,
		GitHub{}.apiBase(host)+"/user/repos?per_page=100&sort=pushed&affiliation=owner,collaborator,organization_member",
		token, nil, &raw)
	if err != nil {
		return nil, err
	}
	out := make([]RepoRef, 0, len(raw))
	for _, r := range raw {
		out = append(out, RepoRef{Path: r.FullName, Name: r.Name, Host: host, Private: r.Private})
	}
	return out, nil
}

func parsePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	return ParseTime(*s)
}
