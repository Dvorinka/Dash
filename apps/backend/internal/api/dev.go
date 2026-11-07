package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdvorak/dash/internal/dev"
	"go.uber.org/zap"
)

// ---------- model ----------

const devProjectCols = `id, provider, host, path, name, token, installation_id,
	enabled, position, last_sync_at, last_sync_error, meta, created_at`

type devProjectRow struct {
	ID             string
	Provider       string
	Host           string
	Path           string
	Name           string
	Token          string
	InstallationID string
	Enabled        bool
	Position       float64
	LastSyncAt     sql.NullString
	LastSyncError  string
	Meta           string
	CreatedAt      string
}

func scanDevProject(row interface{ Scan(...any) error }) (*devProjectRow, error) {
	var p devProjectRow
	var enabled int
	err := row.Scan(&p.ID, &p.Provider, &p.Host, &p.Path, &p.Name, &p.Token,
		&p.InstallationID, &enabled, &p.Position, &p.LastSyncAt, &p.LastSyncError,
		&p.Meta, &p.CreatedAt)
	p.Enabled = enabled != 0
	return &p, err
}

func (p *devProjectRow) project() dev.Project {
	out := dev.Project{
		ID: p.ID, Provider: p.Provider, Host: p.Host, Path: p.Path, Name: p.Name,
		Token: p.Token, InstallationID: p.InstallationID, Enabled: p.Enabled,
		Position: p.Position, LastSyncError: p.LastSyncError,
	}
	if p.LastSyncAt.Valid {
		if t, err := time.Parse(time.RFC3339, p.LastSyncAt.String); err == nil {
			out.LastSyncAt = &t
		}
	}
	if t, err := time.Parse("2006-01-02 15:04:05", p.CreatedAt); err == nil {
		out.CreatedAt = t
	}
	_ = json.Unmarshal([]byte(p.Meta), &out.Meta)
	return out
}

// settingStr reads one settings key as a plain string.
func (s *Server) settingStr(key string) string {
	var v string
	_ = s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	var str string
	if json.Unmarshal([]byte(v), &str) == nil {
		return str
	}
	return ""
}

// devToken resolves the credential for a project: per-project PAT, then the
// provider's global token setting, then a GitHub App installation token.
func (s *Server) devToken(ctx context.Context, p dev.Project) (string, error) {
	if p.Token != "" {
		return p.Token, nil
	}
	if tok := s.settingStr("dev." + p.Provider + "Token"); tok != "" {
		return tok, nil
	}
	if p.Provider == "github" {
		appID := s.settingStr("dev.githubAppId")
		key := s.settingStr("dev.githubAppPrivateKey")
		if appID == "" || key == "" {
			return "", fmt.Errorf("no github token configured")
		}
		inst := p.InstallationID
		if inst == "" {
			inst = s.settingStr("dev.githubAppInstallationId")
		}
		if inst == "" {
			// default to the first installation the app can see
			insts, err := dev.GitHub{}.AppInstallations(ctx, p.Host, appID, key)
			if err != nil || len(insts) == 0 {
				return "", fmt.Errorf("github app: no installations")
			}
			inst = strconv.FormatInt(insts[0].ID, 10)
		}
		return dev.GitHub{}.InstallationToken(ctx, p.Host, appID, key, inst)
	}
	return "", fmt.Errorf("no %s token configured", p.Provider)
}

// ---------- handlers ----------

func (s *Server) listDevProjects(c *gin.Context) {
	rows, err := s.db.Query(`SELECT ` + devProjectCols + ` FROM dev_projects ORDER BY position, created_at`)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []dev.Project{}
	for rows.Next() {
		p, err := scanDevProject(rows)
		if err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		proj := p.project()
		_ = s.db.QueryRow(`SELECT
			(SELECT COUNT(*) FROM dev_issues WHERE project_id=? AND state='open'),
			(SELECT COUNT(*) FROM dev_prs WHERE project_id=? AND state='open')`,
			p.ID, p.ID).Scan(&proj.OpenIssues, &proj.OpenPRs)
		out = append(out, proj)
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) createDevProject(c *gin.Context) {
	var in struct {
		Provider       string `json:"provider"`
		Host           string `json:"host"`
		Path           string `json:"path"`
		Name           string `json:"name"`
		Token          string `json:"token"`
		InstallationID string `json:"installationId"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	in.Path = strings.Trim(strings.TrimSpace(in.Path), "/")
	if in.Provider == "" || in.Path == "" {
		fail(c, http.StatusBadRequest, "provider and path required")
		return
	}
	if _, err := dev.For(in.Provider); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	id := newID("dp")
	name := in.Name
	if name == "" {
		name = in.Path
	}
	if _, err := s.db.Exec(`INSERT INTO dev_projects
		(id, provider, host, path, name, token, installation_id)
		VALUES (?,?,?,?,?,?,?)`,
		id, in.Provider, strings.TrimRight(in.Host, "/"), in.Path, name,
		in.Token, in.InstallationID); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	// Kick an immediate sync in the background.
	go s.syncDevProject(context.Background(), id)
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (s *Server) getDevProject(c *gin.Context) {
	p, ok := s.getDevProjectRow(c)
	if !ok {
		return
	}
	proj := p.project()
	_ = s.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM dev_issues WHERE project_id=? AND state='open'),
		(SELECT COUNT(*) FROM dev_prs WHERE project_id=? AND state='open')`,
		p.ID, p.ID).Scan(&proj.OpenIssues, &proj.OpenPRs)
	c.JSON(http.StatusOK, proj)
}

func (s *Server) getDevProjectRow(c *gin.Context) (*devProjectRow, bool) {
	p, err := scanDevProject(s.db.QueryRow(
		`SELECT `+devProjectCols+` FROM dev_projects WHERE id = ?`, c.Param("id")))
	if err != nil {
		fail(c, http.StatusNotFound, "project not found")
		return nil, false
	}
	return p, true
}

func (s *Server) updateDevProject(c *gin.Context) {
	p, ok := s.getDevProjectRow(c)
	if !ok {
		return
	}
	var in struct {
		Name           *string `json:"name"`
		Token          *string `json:"token"`
		InstallationID *string `json:"installationId"`
		Enabled        *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Name != nil {
		_, _ = s.db.Exec(`UPDATE dev_projects SET name=? WHERE id=?`, *in.Name, p.ID)
	}
	if in.Token != nil {
		_, _ = s.db.Exec(`UPDATE dev_projects SET token=? WHERE id=?`, *in.Token, p.ID)
	}
	if in.InstallationID != nil {
		_, _ = s.db.Exec(`UPDATE dev_projects SET installation_id=? WHERE id=?`, *in.InstallationID, p.ID)
	}
	if in.Enabled != nil {
		en := 0
		if *in.Enabled {
			en = 1
		}
		_, _ = s.db.Exec(`UPDATE dev_projects SET enabled=? WHERE id=?`, en, p.ID)
	}
	if in.Enabled != nil && *in.Enabled {
		go s.syncDevProject(context.Background(), p.ID)
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) deleteDevProject(c *gin.Context) {
	p, ok := s.getDevProjectRow(c)
	if !ok {
		return
	}
	_, _ = s.db.Exec(`DELETE FROM dev_prs WHERE project_id=?`, p.ID)
	_, _ = s.db.Exec(`DELETE FROM dev_issues WHERE project_id=?`, p.ID)
	_, _ = s.db.Exec(`DELETE FROM dev_projects WHERE id=?`, p.ID)
	c.Status(http.StatusNoContent)
}

func (s *Server) syncDevProjectNow(c *gin.Context) {
	p, ok := s.getDevProjectRow(c)
	if !ok {
		return
	}
	go s.syncDevProject(context.Background(), p.ID)
	c.Status(http.StatusAccepted)
}

// syncDevProject pulls meta + issues + PRs for one project and upserts.
func (s *Server) syncDevProject(ctx context.Context, id string) {
	row, err := scanDevProject(s.db.QueryRow(
		`SELECT `+devProjectCols+` FROM dev_projects WHERE id=?`, id))
	if err != nil {
		return
	}
	p := row.project()
	provider, err := dev.For(p.Provider)
	if err != nil {
		return
	}
	token, err := s.devToken(ctx, p)
	if err != nil {
		s.setSyncErr(id, err)
		return
	}
	meta, err := provider.Project(ctx, p, token)
	if err != nil {
		s.setSyncErr(id, err)
		return
	}
	metaJSON, _ := json.Marshal(meta)
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = s.db.Exec(`UPDATE dev_projects SET meta=?, last_sync_at=?, last_sync_error='' WHERE id=?`,
		string(metaJSON), now, id)

	issues, err := provider.Issues(ctx, p, token)
	if err == nil {
		s.upsertIssues(id, issues)
	}
	prs, err := provider.PullRequests(ctx, p, token)
	if err == nil {
		s.upsertPRs(id, prs)
	}
}

func (s *Server) setSyncErr(id string, err error) {
	msg := err.Error()
	if len(msg) > 300 {
		msg = msg[:300]
	}
	_, _ = s.db.Exec(`UPDATE dev_projects SET last_sync_error=?, last_sync_at=? WHERE id=?`,
		msg, time.Now().UTC().Format(time.RFC3339), id)
	s.log.Warn("dev project sync failed", zap.String("id", id), zap.Error(err))
}

func (s *Server) upsertIssues(projectID string, issues []dev.Issue) {
	var firstErr error
	for _, i := range issues {
		labels, _ := json.Marshal(i.Labels)
		assignees, _ := json.Marshal(i.Assignees)
		var closedAt any
		if i.ClosedAt != nil {
			closedAt = i.ClosedAt.UTC().Format(time.RFC3339)
		}
		_, err := s.db.Exec(`INSERT INTO dev_issues
			(project_id, number, iid, title, state, author, assignees, labels, url, body, comments,
			 created_at, updated_at, closed_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT(project_id, number) DO UPDATE SET
			  iid=excluded.iid, title=excluded.title, state=excluded.state,
			  author=excluded.author, assignees=excluded.assignees,
			  labels=excluded.labels, url=excluded.url,
			  body=excluded.body, comments=excluded.comments,
			  updated_at=excluded.updated_at, closed_at=excluded.closed_at`,
			projectID, i.Number, i.IID, i.Title, i.State, i.Author, string(assignees), string(labels),
			i.URL, i.Body, i.Comments,
			i.CreatedAt.UTC().Format(time.RFC3339), i.UpdatedAt.UTC().Format(time.RFC3339), closedAt)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		s.log.Warn("dev issue upsert failed", zap.String("project", projectID), zap.Error(firstErr))
	}
}

func (s *Server) upsertPRs(projectID string, prs []dev.PullRequest) {
	var firstErr error
	for _, pr := range prs {
		var mergedAt, closedAt any
		if pr.MergedAt != nil {
			mergedAt = pr.MergedAt.UTC().Format(time.RFC3339)
		}
		if pr.ClosedAt != nil {
			closedAt = pr.ClosedAt.UTC().Format(time.RFC3339)
		}
		draft := 0
		if pr.Draft {
			draft = 1
		}
		assignees, _ := json.Marshal(pr.Assignees)
		reviewers, _ := json.Marshal(pr.Reviewers)
		_, err := s.db.Exec(`INSERT INTO dev_prs
			(project_id, number, title, state, draft, author, assignees, reviewers,
			 source_branch, target_branch,
			 additions, deletions, changed_files, mergeable, review_decision, url, body,
			 comments, created_at, updated_at, merged_at, closed_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT(project_id, number) DO UPDATE SET
			  title=excluded.title, state=excluded.state, draft=excluded.draft,
			  author=excluded.author, assignees=excluded.assignees,
			  reviewers=excluded.reviewers,
			  source_branch=excluded.source_branch,
			  target_branch=excluded.target_branch, additions=excluded.additions,
			  deletions=excluded.deletions, changed_files=excluded.changed_files,
			  mergeable=excluded.mergeable, review_decision=excluded.review_decision,
			  url=excluded.url, body=excluded.body, comments=excluded.comments,
			  updated_at=excluded.updated_at, merged_at=excluded.merged_at,
			  closed_at=excluded.closed_at`,
			projectID, pr.Number, pr.Title, pr.State, draft, pr.Author,
			string(assignees), string(reviewers),
			pr.SourceBranch, pr.TargetBranch, pr.Additions, pr.Deletions, pr.ChangedFiles,
			pr.Mergeable, pr.ReviewDecision, pr.URL, pr.Body, pr.Comments,
			pr.CreatedAt.UTC().Format(time.RFC3339), pr.UpdatedAt.UTC().Format(time.RFC3339),
			mergedAt, closedAt)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		s.log.Warn("dev upsert failed", zap.String("project", projectID), zap.Error(firstErr))
	}
}

// ---------- read endpoints ----------

func (s *Server) listDevIssues(c *gin.Context) {
	pid := c.Param("id")
	state := c.DefaultQuery("state", "open")
	limit := 200
	if v, err := strconv.Atoi(c.DefaultQuery("limit", "200")); err == nil && v > 0 && v <= 500 {
		limit = v
	}
	q := `SELECT number, iid, title, state, author, assignees, labels, url, body, comments,
		created_at, updated_at, closed_at FROM dev_issues WHERE project_id=?`
	args := []any{pid}
	if state != "all" {
		q += ` AND state=?`
		args = append(args, state)
	}
	q += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var number, iid, comments int
		var title, state, author, assignees, labels, url, body string
		var created, updated, closed sql.NullString
		if err := rows.Scan(&number, &iid, &title, &state, &author, &assignees, &labels, &url, &body,
			&comments, &created, &updated, &closed); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		var lbs, people []string
		_ = json.Unmarshal([]byte(labels), &lbs)
		_ = json.Unmarshal([]byte(assignees), &people)
		out = append(out, gin.H{
			"number": number, "iid": iid, "title": title, "state": state,
			"author": author, "assignees": people, "labels": lbs, "url": url, "body": body,
			"comments": comments, "createdAt": created.String,
			"updatedAt": updated.String, "closedAt": nullStr(closed),
		})
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) listDevPRs(c *gin.Context) {
	pid := c.Param("id")
	state := c.DefaultQuery("state", "open")
	limit := 200
	if v, err := strconv.Atoi(c.DefaultQuery("limit", "200")); err == nil && v > 0 && v <= 500 {
		limit = v
	}
	q := `SELECT number, title, state, draft, author, assignees, reviewers,
		source_branch, target_branch,
		additions, deletions, changed_files, mergeable, review_decision, url, body,
		comments, created_at, updated_at, merged_at, closed_at
		FROM dev_prs WHERE project_id=?`
	args := []any{pid}
	if state != "all" {
		q += ` AND state=?`
		args = append(args, state)
	}
	q += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var number, draft, additions, deletions, changed, comments int
		var title, state, author, assignees, reviewers, src, dst, mergeable, review, url, body string
		var created, updated, merged, closed sql.NullString
		if err := rows.Scan(&number, &title, &state, &draft, &author, &assignees, &reviewers, &src, &dst,
			&additions, &deletions, &changed, &mergeable, &review, &url, &body,
			&comments, &created, &updated, &merged, &closed); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		var people, revs []string
		_ = json.Unmarshal([]byte(assignees), &people)
		_ = json.Unmarshal([]byte(reviewers), &revs)
		out = append(out, gin.H{
			"number": number, "title": title, "state": state, "draft": draft != 0,
			"author": author, "assignees": people, "reviewers": revs,
			"sourceBranch": src, "targetBranch": dst,
			"additions": additions, "deletions": deletions, "changedFiles": changed,
			"mergeable": mergeable, "reviewDecision": review, "url": url, "body": body,
			"comments": comments, "createdAt": created.String,
			"updatedAt": updated.String, "mergedAt": nullStr(merged),
			"closedAt": nullStr(closed),
		})
	}
	c.JSON(http.StatusOK, out)
}

func nullStr(ns sql.NullString) any {
	if ns.Valid {
		return ns.String
	}
	return nil
}

// devOverview aggregates counts + a cross-project activity feed.
func (s *Server) devOverview(c *gin.Context) {
	var projects, openIssues, openPRs, merged int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM dev_projects WHERE enabled=1`).Scan(&projects)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM dev_issues WHERE state='open'`).Scan(&openIssues)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM dev_prs WHERE state='open'`).Scan(&openPRs)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM dev_prs WHERE state='merged'`).Scan(&merged)

	type feedItem struct {
		ProjectID string `json:"projectId"`
		Kind      string `json:"kind"` // issue | pr
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		Author    string `json:"author"`
		UpdatedAt string `json:"updatedAt"`
		URL       string `json:"url"`
	}
	feed := []feedItem{}
	rows, err := s.db.Query(`SELECT project_id, 'issue', number, title, state, author, updated_at, url
		FROM dev_issues UNION ALL
		SELECT project_id, 'pr', number, title, state, author, updated_at, url
		FROM dev_prs ORDER BY updated_at DESC LIMIT 50`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var f feedItem
			var upd sql.NullString
			if err := rows.Scan(&f.ProjectID, &f.Kind, &f.Number, &f.Title,
				&f.State, &f.Author, &upd, &f.URL); err == nil {
				f.UpdatedAt = upd.String
				feed = append(feed, f)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"projects": projects, "openIssues": openIssues,
		"openPRs": openPRs, "mergedPRs": merged, "activity": feed,
	})
}

// devContributions serves the heatmap; cached 5min per project.
var devContribCache sync.Map // projectID -> {at, days}

func (s *Server) devContributions(c *gin.Context) {
	p, ok := s.getDevProjectRow(c)
	if !ok {
		return
	}
	pid := p.ID
	if v, ok := devContribCache.Load(pid); ok {
		ent := v.(struct {
			at   time.Time
			days []dev.DayCount
		})
		if time.Since(ent.at) < 5*time.Minute {
			c.JSON(http.StatusOK, ent.days)
			return
		}
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	provider, err := dev.For(p.Provider)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	token, err := s.devToken(ctx, p.project())
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	days, err := provider.Contributions(ctx, p.project(), token)
	if err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	devContribCache.Store(pid, struct {
		at   time.Time
		days []dev.DayCount
	}{time.Now(), days})
	c.JSON(http.StatusOK, days)
}

// ---------- actions ----------

type devAction struct {
	Provider dev.Provider
	Project  dev.Project
	Token    string
}

func (s *Server) devActionFor(c *gin.Context) (*devAction, bool) {
	p, ok := s.getDevProjectRow(c)
	if !ok {
		return nil, false
	}
	provider, err := dev.For(p.Provider)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return nil, false
	}
	token, err := s.devToken(c.Request.Context(), p.project())
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return nil, false
	}
	return &devAction{Provider: provider, Project: p.project(), Token: token}, true
}

func (s *Server) devMergePR(c *gin.Context) {
	a, ok := s.devActionFor(c)
	if !ok {
		return
	}
	n, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		fail(c, http.StatusBadRequest, "bad number")
		return
	}
	var in struct {
		Method string `json:"method"`
	}
	_ = c.ShouldBindJSON(&in)
	if err := a.Provider.MergePR(c.Request.Context(), a.Project, a.Token, n, in.Method); err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	go s.syncDevProject(context.Background(), a.Project.ID)
	c.Status(http.StatusNoContent)
}

func (s *Server) devClosePR(c *gin.Context) {
	a, ok := s.devActionFor(c)
	if !ok {
		return
	}
	n, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		fail(c, http.StatusBadRequest, "bad number")
		return
	}
	if err := a.Provider.ClosePR(c.Request.Context(), a.Project, a.Token, n); err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	go s.syncDevProject(context.Background(), a.Project.ID)
	c.Status(http.StatusNoContent)
}

func (s *Server) devCloseIssue(c *gin.Context) {
	a, ok := s.devActionFor(c)
	if !ok {
		return
	}
	n, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		fail(c, http.StatusBadRequest, "bad number")
		return
	}
	if err := a.Provider.CloseIssue(c.Request.Context(), a.Project, a.Token, n); err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	go s.syncDevProject(context.Background(), a.Project.ID)
	c.Status(http.StatusNoContent)
}

// devItemAction dispatches one provider call against an issue or PR number.
// needsBody requires a non-empty JSON {body} (comments).
func (s *Server) devItemAction(c *gin.Context, needsBody bool, call func(a *devAction, n int, body string) error) {
	a, ok := s.devActionFor(c)
	if !ok {
		return
	}
	n, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		fail(c, http.StatusBadRequest, "bad number")
		return
	}
	var in struct {
		Body   string `json:"body"`
		Method string `json:"method"`
	}
	_ = c.ShouldBindJSON(&in)
	if needsBody && strings.TrimSpace(in.Body) == "" {
		fail(c, http.StatusBadRequest, "body required")
		return
	}
	if err := call(a, n, in.Body); err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	go s.syncDevProject(context.Background(), a.Project.ID)
	c.Status(http.StatusNoContent)
}

func (s *Server) devReopenIssue(c *gin.Context) {
	s.devItemAction(c, false, func(a *devAction, n int, _ string) error {
		return a.Provider.ReopenIssue(c.Request.Context(), a.Project, a.Token, n)
	})
}

func (s *Server) devReopenPR(c *gin.Context) {
	s.devItemAction(c, false, func(a *devAction, n int, _ string) error {
		return a.Provider.ReopenPR(c.Request.Context(), a.Project, a.Token, n)
	})
}

func (s *Server) devCommentIssue(c *gin.Context) {
	s.devItemAction(c, true, func(a *devAction, n int, body string) error {
		return a.Provider.CommentIssue(c.Request.Context(), a.Project, a.Token, n, body)
	})
}

func (s *Server) devCommentPR(c *gin.Context) {
	s.devItemAction(c, true, func(a *devAction, n int, body string) error {
		return a.Provider.CommentPR(c.Request.Context(), a.Project, a.Token, n, body)
	})
}

// devSyncAll triggers a sync for every enabled project.
func (s *Server) devSyncAll(c *gin.Context) {
	rows, err := s.db.Query(`SELECT id FROM dev_projects WHERE enabled=1`)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		go s.syncDevProject(context.Background(), id)
	}
	c.Status(http.StatusAccepted)
}

// devComments lists comments on an issue or PR (kind: "issue" | "pr").
func (s *Server) devComments(c *gin.Context, kind string) {
	a, ok := s.devActionFor(c)
	if !ok {
		return
	}
	n, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		fail(c, http.StatusBadRequest, "bad number")
		return
	}
	comments, err := a.Provider.Comments(c.Request.Context(), a.Project, a.Token, kind, n)
	if err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	c.JSON(http.StatusOK, comments)
}

func (s *Server) devIssueComments(c *gin.Context) { s.devComments(c, "issue") }
func (s *Server) devPRComments(c *gin.Context)    { s.devComments(c, "pr") }

func (s *Server) devPRFiles(c *gin.Context) {
	a, ok := s.devActionFor(c)
	if !ok {
		return
	}
	n, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		fail(c, http.StatusBadRequest, "bad number")
		return
	}
	files, err := a.Provider.PRFiles(c.Request.Context(), a.Project, a.Token, n)
	if err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	c.JSON(http.StatusOK, files)
}

// devPerson handles {username, on} bodies for assignee/reviewer toggles.
func devPerson(c *gin.Context) (string, bool, bool) {
	var in struct {
		Username string `json:"username"`
		On       *bool  `json:"on"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Username == "" || in.On == nil {
		return "", false, false
	}
	return in.Username, *in.On, true
}

func (s *Server) devAssignee(c *gin.Context, kind string) {
	a, ok := s.devActionFor(c)
	if !ok {
		return
	}
	n, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		fail(c, http.StatusBadRequest, "bad number")
		return
	}
	username, on, ok2 := devPerson(c)
	if !ok2 {
		fail(c, http.StatusBadRequest, "username and on required")
		return
	}
	if err := a.Provider.SetAssignee(c.Request.Context(), a.Project, a.Token, kind, n, username, on); err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	go s.syncDevProject(context.Background(), a.Project.ID)
	c.Status(http.StatusNoContent)
}

func (s *Server) devIssueAssignee(c *gin.Context) { s.devAssignee(c, "issue") }
func (s *Server) devPRAssignee(c *gin.Context)    { s.devAssignee(c, "pr") }

func (s *Server) devReviewer(c *gin.Context) {
	a, ok := s.devActionFor(c)
	if !ok {
		return
	}
	n, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		fail(c, http.StatusBadRequest, "bad number")
		return
	}
	username, on, ok2 := devPerson(c)
	if !ok2 {
		fail(c, http.StatusBadRequest, "username and on required")
		return
	}
	if err := a.Provider.SetReviewer(c.Request.Context(), a.Project, a.Token, n, username, on); err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	go s.syncDevProject(context.Background(), a.Project.ID)
	c.Status(http.StatusNoContent)
}

// devDiscover lists repos a token/app can access (auto-detect).
func (s *Server) devDiscover(c *gin.Context) {
	var in struct {
		Provider       string `json:"provider"`
		Host           string `json:"host"`
		Token          string `json:"token"`
		InstallationID string `json:"installationId"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Provider == "" {
		fail(c, http.StatusBadRequest, "provider required")
		return
	}
	provider, err := dev.For(in.Provider)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	token := in.Token
	if token == "" {
		token = s.settingStr("dev." + in.Provider + "Token")
	}
	if token == "" && in.Provider == "github" {
		appID := s.settingStr("dev.githubAppId")
		key := s.settingStr("dev.githubAppPrivateKey")
		if appID == "" || key == "" {
			fail(c, http.StatusBadRequest, "no token or github app configured")
			return
		}
		insts, err := dev.GitHub{}.AppInstallations(c.Request.Context(), in.Host, appID, key)
		if err != nil {
			fail(c, http.StatusBadGateway, err.Error())
			return
		}
		if len(insts) == 0 {
			c.JSON(http.StatusOK, []any{})
			return
		}
		refs, err := dev.GitHub{}.AppDiscover(c.Request.Context(), in.Host, appID, key,
			strconv.FormatInt(insts[0].ID, 10))
		if err != nil {
			fail(c, http.StatusBadGateway, err.Error())
			return
		}
		c.JSON(http.StatusOK, refs)
		return
	}
	if token == "" {
		fail(c, http.StatusBadRequest, "token required")
		return
	}
	refs, err := provider.Discover(c.Request.Context(), in.Host, token)
	if err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	c.JSON(http.StatusOK, refs)
}

// devAppInstallations lists GitHub App installations for the setup UI.
func (s *Server) devAppInstallations(c *gin.Context) {
	appID := s.settingStr("dev.githubAppId")
	key := s.settingStr("dev.githubAppPrivateKey")
	if appID == "" || key == "" {
		c.JSON(http.StatusOK, []any{})
		return
	}
	insts, err := dev.GitHub{}.AppInstallations(c.Request.Context(), "", appID, key)
	if err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	c.JSON(http.StatusOK, insts)
}

// ---------- scheduler ----------

// runDevSync periodically syncs enabled projects whose interval elapsed.
func (s *Server) runDevSync(done <-chan struct{}) {
	tick := time.NewTicker(60 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-done:
			return
		case <-tick.C:
			interval := 300
			if v := s.settingStr("dev.syncIntervalSec"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n >= 30 {
					interval = n
				}
			}
			rows, err := s.db.Query(`SELECT id, last_sync_at FROM dev_projects WHERE enabled=1`)
			if err != nil {
				continue
			}
			type due struct{ id string }
			var ids []string
			for rows.Next() {
				var id string
				var last sql.NullString
				if rows.Scan(&id, &last) == nil {
					if !last.Valid {
						ids = append(ids, id)
						continue
					}
					if t, err := time.Parse(time.RFC3339, last.String); err == nil &&
						time.Since(t) > time.Duration(interval)*time.Second {
						ids = append(ids, id)
					}
				}
			}
			rows.Close()
			for _, id := range ids {
				go s.syncDevProject(context.Background(), id)
			}
		}
	}
}
