package dev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubProject(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("missing bearer token")
		}
		_, _ = w.Write([]byte(`{"description":"d","default_branch":"main","language":"Go","stargazers_count":42,"forks_count":3,"open_issues_count":7,"visibility":"public","homepage":"https://x","pushed_at":"2026-01-01T00:00:00Z"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	meta, err := (GitHub{}).Project(context.Background(),
		Project{Provider: "github", Host: srv.URL, Path: "o/r"}, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Stars != 42 || meta.Language != "Go" || meta.DefaultBranch != "main" {
		t.Fatalf("bad meta: %+v", meta)
	}
}

func TestGitHubIssuesSkipsPRs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/o/r/issues", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"number":1,"title":"bug","state":"open","user":{"login":"ann"},"labels":[{"name":"bug"}],"html_url":"u1","comments":2,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"},
			{"number":2,"title":"pr-as-issue","state":"open","pull_request":{},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}
		]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	issues, err := (GitHub{}).Issues(context.Background(),
		Project{Provider: "github", Host: srv.URL, Path: "o/r"}, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Number != 1 || issues[0].Author != "ann" || len(issues[0].Labels) != 1 {
		t.Fatalf("bad issues: %+v", issues)
	}
}

func TestGitHubPRsMerged(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/o/r/pulls", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"number":5,"title":"feat","state":"closed","merged_at":"2026-01-03T00:00:00Z","draft":false,
			 "user":{"login":"bob"},"head":{"ref":"feat/x"},"base":{"ref":"main"},
			 "additions":10,"deletions":2,"changed_files":3,"mergeable":true,"mergeable_state":"clean",
			 "html_url":"u","comments":0,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-03T00:00:00Z"}
		]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	prs, err := (GitHub{}).PullRequests(context.Background(),
		Project{Provider: "github", Host: srv.URL, Path: "o/r"}, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != 1 || prs[0].State != "merged" || prs[0].Mergeable != "mergeable" || prs[0].Additions != 10 {
		t.Fatalf("bad prs: %+v", prs)
	}
}

func TestGitHubContributions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/graphql", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Query == "" {
			t.Error("empty graphql query")
		}
		_, _ = w.Write([]byte(`{"data":{"user":{"contributionsCollection":{"contributionCalendar":{"totalContributions":5,"weeks":[{"contributionDays":[{"contributionCount":3,"date":"2026-01-01"},{"contributionCount":2,"date":"2026-01-02"}]}]}}}}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	days, err := (GitHub{}).Contributions(context.Background(),
		Project{Provider: "github", Host: srv.URL, Path: "octocat/repo"}, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 2 || days[0].Count != 3 {
		t.Fatalf("bad days: %+v", days)
	}
}

func TestGitLabIssuesAndMRs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/g%2Fp/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "glpat" {
			t.Errorf("missing private token")
		}
		_, _ = w.Write([]byte(`[{"iid":3,"title":"it","state":"opened","author":{"username":"zed"},"labels":[{"title":"a"}],"web_url":"u","user_notes_count":1,"created_at":"2026-01-01T00:00:00.000Z","updated_at":"2026-01-02T00:00:00.000Z","closed_at":null}]`))
	})
	mux.HandleFunc("/api/v4/projects/g%2Fp/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"iid":9,"title":"mr","state":"opened","draft":true,"author":{"username":"zed"},"source_branch":"f","target_branch":"main","changes_count":"4","user_notes_count":0,"web_url":"u","merge_status":"can_be_merged","created_at":"2026-01-01T00:00:00.000Z","updated_at":"2026-01-02T00:00:00.000Z"}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := Project{Provider: "gitlab", Host: srv.URL, Path: "g/p"}
	issues, err := (GitLab{}).Issues(context.Background(), p, "glpat")
	if err != nil || len(issues) != 1 || issues[0].Number != 3 || issues[0].State != "open" {
		t.Fatalf("bad issues: %v %+v", err, issues)
	}
	prs, err := (GitLab{}).PullRequests(context.Background(), p, "glpat")
	if err != nil || len(prs) != 1 || prs[0].Number != 9 || prs[0].State != "open" || !prs[0].Draft {
		t.Fatalf("bad prs: %v %+v", err, prs)
	}
}

func TestGitLabContributions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/g%2Fp/repository/commits", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"authored_date":"2026-01-01T10:00:00.000Z"},
			{"authored_date":"2026-01-01T11:00:00.000Z"},
			{"authored_date":"2026-01-03T09:00:00.000Z"}
		]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	days, err := (GitLab{}).Contributions(context.Background(),
		Project{Provider: "gitlab", Host: srv.URL, Path: "g/p"}, "tok")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, d := range days {
		counts[d.Date] = d.Count
	}
	if counts["2026-01-01"] != 2 || counts["2026-01-03"] != 1 {
		t.Fatalf("bad days: %+v", days)
	}
}
