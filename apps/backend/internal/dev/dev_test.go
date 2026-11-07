package dev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGiteaProvider(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token gt" {
			t.Errorf("missing token auth: %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"description":"gt","default_branch":"main","language":"Go","stargazers_count":3,"forks_count":1,"open_issues_count":2,"visibility":"public","updated_at":"2026-11-01T00:00:00Z"}`))
	})
	mux.HandleFunc("/api/v1/repos/o/r/issues", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"index":5,"title":"real issue","state":"open","user":{"login":"zed"},"labels":[{"name":"bug"}],"html_url":"u","comments":1,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"},
			{"index":6,"title":"leaked pr","state":"open","pull_request":{},"user":{"login":"zed"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"}
		]`))
	})
	mux.HandleFunc("/api/v1/repos/o/r/pulls", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"index":7,"title":"gt pr","state":"open","draft":false,"merged":false,"user":{"login":"zed"},"head":{"ref":"f"},"base":{"ref":"main"},"additions":5,"deletions":2,"changed_files":1,"mergeable":true,"html_url":"u","comments":0,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"},
			{"index":8,"title":"gt merged","state":"closed","merged":true,"merged_at":"2026-01-05T00:00:00Z","user":{"login":"zed"},"head":{"ref":"g"},"base":{"ref":"main"},"mergeable":null,"html_url":"u","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-05T00:00:00Z"}
		]`))
	})
	mux.HandleFunc("/api/v1/repos/o/r/commits", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"commit":{"author":{"date":"2026-01-02T10:00:00Z"}}}]`))
	})
	mux.HandleFunc("/api/v1/user/repos", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"full_name":"o/r","name":"r","private":false}]`))
	})
	var lastMethod, lastPath string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		lastMethod, lastPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := Project{Provider: "gitea", Host: srv.URL, Path: "o/r"}
	meta, err := (Gitea{}).Project(context.Background(), p, "gt")
	if err != nil || meta.Stars != 3 || meta.Language != "Go" {
		t.Fatalf("bad meta: %v %+v", err, meta)
	}
	issues, err := (Gitea{}).Issues(context.Background(), p, "gt")
	if err != nil || len(issues) != 1 || issues[0].Number != 5 {
		t.Fatalf("bad issues: %v %+v", err, issues)
	}
	prs, err := (Gitea{}).PullRequests(context.Background(), p, "gt")
	if err != nil || len(prs) != 2 || prs[0].State != "open" || prs[1].State != "merged" {
		t.Fatalf("bad prs: %v %+v", err, prs)
	}
	days, err := (Gitea{}).Contributions(context.Background(), p, "gt")
	if err != nil || len(days) != 1 || days[0].Date != "2026-01-02" {
		t.Fatalf("bad days: %v %+v", err, days)
	}
	refs, err := (Gitea{}).Discover(context.Background(), srv.URL, "gt")
	if err != nil || len(refs) != 1 || refs[0].Path != "o/r" {
		t.Fatalf("bad discover: %v %+v", err, refs)
	}
	if _, err := For("gitea"); err != nil {
		t.Fatal(err)
	}
	// action endpoints hit the catch-all and return 204
	if err := (Gitea{}).ReopenIssue(context.Background(), p, "gt", 5); err != nil {
		t.Fatal(err)
	}
	if err := (Gitea{}).CommentPR(context.Background(), p, "gt", 7, "hi"); err != nil {
		t.Fatal(err)
	}
	_ = lastMethod
	_ = lastPath
}

func TestGitHubCommentsAndFiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/o/r/issues/11/comments", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"user":{"login":"alice"},"body":"hi","created_at":"2026-01-02T00:00:00Z","html_url":"u"}]`))
	})
	mux.HandleFunc("/api/v3/repos/o/r/pulls/21/files", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"filename":"a.go","status":"modified","additions":3,"deletions":1,"patch":"@@ -1 +1 @@\n-old\n+new"}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	p := Project{Provider: "github", Host: srv.URL, Path: "o/r"}

	cs, err := (GitHub{}).Comments(context.Background(), p, "t", "issue", 11)
	if err != nil || len(cs) != 1 || cs[0].Author != "alice" || cs[0].Body != "hi" {
		t.Fatalf("bad comments: %v %+v", err, cs)
	}
	fs, err := (GitHub{}).PRFiles(context.Background(), p, "t", 21)
	if err != nil || len(fs) != 1 || fs[0].Filename != "a.go" || fs[0].Additions != 3 || !strings.Contains(fs[0].Patch, "+new") {
		t.Fatalf("bad files: %v %+v", err, fs)
	}
}

func TestGitLabCommentsAndFiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/g%2Fp/merge_requests/9/notes", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"author":{"username":"zed"},"body":"real","system":false,"created_at":"2026-01-02T00:00:00.000Z"},
			{"author":{"username":"bot"},"body":"sys","system":true,"created_at":"2026-01-02T00:00:00.000Z"}
		]`))
	})
	mux.HandleFunc("/api/v4/projects/g%2Fp/merge_requests/9/diffs", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"old_path":"a","new_path":"b.go","new_file":true,"diff":"@@\n+one\n+two\n-three\n"}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	p := Project{Provider: "gitlab", Host: srv.URL, Path: "g/p"}

	cs, err := (GitLab{}).Comments(context.Background(), p, "t", "pr", 9)
	if err != nil || len(cs) != 1 || cs[0].Author != "zed" {
		t.Fatalf("bad comments: %v %+v", err, cs)
	}
	fs, err := (GitLab{}).PRFiles(context.Background(), p, "t", 9)
	if err != nil || len(fs) != 1 || fs[0].Filename != "b.go" || fs[0].Status != "added" || fs[0].Additions != 2 || fs[0].Deletions != 1 {
		t.Fatalf("bad files: %v %+v", err, fs)
	}
}
