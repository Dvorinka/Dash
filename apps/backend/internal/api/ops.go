package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ---------- incidents ----------

type Incident struct {
	ID        string           `json:"id"`
	Title     string           `json:"title"`
	Severity  string           `json:"severity"`
	Status    string           `json:"status"`
	MonitorID *string          `json:"monitorId"`
	Updates   []IncidentUpdate `json:"updates,omitempty"`
	CreatedAt string           `json:"createdAt"`
	UpdatedAt string           `json:"updatedAt"`
}

type IncidentUpdate struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	At        string `json:"at"`
}

var incidentStatuses = map[string]bool{"open": true, "ack": true, "resolved": true, "closed": true}
var incidentSeverities = map[string]bool{"minor": true, "major": true, "critical": true}

func scanIncident(row interface{ Scan(...any) error }) (*Incident, error) {
	var in Incident
	err := row.Scan(&in.ID, &in.Title, &in.Severity, &in.Status, &in.MonitorID,
		&in.CreatedAt, &in.UpdatedAt)
	return &in, err
}

const incidentCols = `id, title, severity, status, monitor_id, created_at, updated_at`

func (s *Server) loadIncident(id string) (*Incident, error) {
	return scanIncident(s.db.QueryRow(`SELECT `+incidentCols+` FROM incidents WHERE id = ?`, id))
}

func (s *Server) incidentUpdates(id string) []IncidentUpdate {
	rows, err := s.db.Query(`SELECT id, status, message, at FROM incident_updates
		WHERE incident_id = ? ORDER BY at`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []IncidentUpdate{}
	for rows.Next() {
		var u IncidentUpdate
		if rows.Scan(&u.ID, &u.Status, &u.Message, &u.At) == nil {
			out = append(out, u)
		}
	}
	return out
}

func (s *Server) listIncidents(c *gin.Context) {
	openOnly := c.Query("status") == "open"
	q := `SELECT ` + incidentCols + ` FROM incidents`
	if openOnly {
		q += ` WHERE status IN ('open','ack')`
	}
	q += ` ORDER BY created_at DESC LIMIT 200`
	rows, err := s.db.Query(q)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []Incident{}
	for rows.Next() {
		in, err := scanIncident(rows)
		if err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		in.Updates = s.incidentUpdates(in.ID)
		out = append(out, *in)
	}
	c.JSON(http.StatusOK, out)
}

type incidentInput struct {
	Title     *string `json:"title"`
	Severity  *string `json:"severity"`
	Status    *string `json:"status"`
	MonitorID *string `json:"monitorId"`
	Message   *string `json:"message"` // initial update body / transition note
}

func (s *Server) createIncident(c *gin.Context) {
	var in incidentInput
	if err := c.ShouldBindJSON(&in); err != nil || in.Title == nil || *in.Title == "" {
		fail(c, http.StatusBadRequest, "title required")
		return
	}
	sev := strOr2(in.Severity, "major")
	st := strOr2(in.Status, "open")
	if !incidentSeverities[sev] || !incidentStatuses[st] {
		fail(c, http.StatusBadRequest, "invalid severity or status")
		return
	}
	id := newID("inc")
	_, err := s.db.Exec(`INSERT INTO incidents (id, title, severity, status, monitor_id)
		VALUES (?,?,?,?,?)`, id, *in.Title, sev, st, in.MonitorID)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if msg := strOr(in.Message); msg != "" {
		_, _ = s.db.Exec(`INSERT INTO incident_updates (incident_id, status, message) VALUES (?,?,?)`,
			id, st, msg)
	}
	inc, err := s.loadIncident(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	inc.Updates = s.incidentUpdates(id)
	c.JSON(http.StatusCreated, inc)
}

// patchIncident moves status/severity/title; a status change also appends an
// incident_updates row so the timeline stays honest.
func (s *Server) patchIncident(c *gin.Context) {
	id := c.Param("id")
	cur, err := s.loadIncident(id)
	if notFound(err) {
		fail(c, http.StatusNotFound, "incident not found")
		return
	}
	var in incidentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Severity != nil && !incidentSeverities[*in.Severity] {
		fail(c, http.StatusBadRequest, "invalid severity")
		return
	}
	if in.Status != nil && !incidentStatuses[*in.Status] {
		fail(c, http.StatusBadRequest, "invalid status")
		return
	}
	set := func(col string, p *string) {
		if p != nil {
			_, _ = s.db.Exec(`UPDATE incidents SET `+col+` = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`, *p, id)
		}
	}
	set("title", in.Title)
	set("severity", in.Severity)
	set("status", in.Status)
	if in.Status != nil && *in.Status != cur.Status {
		_, _ = s.db.Exec(`INSERT INTO incident_updates (incident_id, status, message) VALUES (?,?,?)`,
			id, *in.Status, strOr(in.Message))
	}
	inc, err := s.loadIncident(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	inc.Updates = s.incidentUpdates(id)
	c.JSON(http.StatusOK, inc)
}

func (s *Server) addIncidentUpdate(c *gin.Context) {
	id := c.Param("id")
	if _, err := s.loadIncident(id); notFound(err) {
		fail(c, http.StatusNotFound, "incident not found")
		return
	}
	var in incidentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	st := strOr2(in.Status, "")
	if st != "" && !incidentStatuses[st] {
		fail(c, http.StatusBadRequest, "invalid status")
		return
	}
	if st == "" {
		st = "note"
	}
	_, err := s.db.Exec(`INSERT INTO incident_updates (incident_id, status, message) VALUES (?,?,?)`,
		id, st, strOr(in.Message))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if in.Status != nil {
		_, _ = s.db.Exec(`UPDATE incidents SET status = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`, *in.Status, id)
	}
	inc, _ := s.loadIncident(id)
	inc.Updates = s.incidentUpdates(id)
	c.JSON(http.StatusOK, inc)
}

func (s *Server) deleteIncident(c *gin.Context) {
	res, err := s.db.Exec(`DELETE FROM incidents WHERE id = ?`, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "incident not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------- maintenance windows ----------

type MaintWindow struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	StartsAt   string   `json:"startsAt"`
	EndsAt     string   `json:"endsAt"`
	MonitorIDs []string `json:"monitorIds"` // empty = all monitors
	Active     bool     `json:"active"`
	CreatedAt  string   `json:"createdAt"`
}

func scanWindow(row interface{ Scan(...any) error }) (*MaintWindow, error) {
	var w MaintWindow
	var ids string
	err := row.Scan(&w.ID, &w.Title, &w.StartsAt, &w.EndsAt, &ids, &w.CreatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(ids), &w.MonitorIDs)
	if w.MonitorIDs == nil {
		w.MonitorIDs = []string{}
	}
	now := time.Now().UTC()
	if st, err1 := time.Parse(time.RFC3339Nano, w.StartsAt); err1 == nil {
		if en, err2 := time.Parse(time.RFC3339Nano, w.EndsAt); err2 == nil {
			w.Active = !now.Before(st) && now.Before(en)
		}
	}
	return &w, nil
}

const windowCols = `id, title, starts_at, ends_at, monitor_ids, created_at`

func (s *Server) listWindows(c *gin.Context) {
	rows, err := s.db.Query(`SELECT ` + windowCols + ` FROM maintenance_windows ORDER BY starts_at DESC LIMIT 100`)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []MaintWindow{}
	for rows.Next() {
		w, err := scanWindow(rows)
		if err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, *w)
	}
	c.JSON(http.StatusOK, out)
}

type windowInput struct {
	Title      *string  `json:"title"`
	StartsAt   *string  `json:"startsAt"`
	EndsAt     *string  `json:"endsAt"`
	MonitorIDs []string `json:"monitorIds"`
}

func validWindow(in *windowInput) bool {
	if in.Title == nil || in.StartsAt == nil || in.EndsAt == nil || *in.Title == "" {
		return false
	}
	st, err1 := time.Parse(time.RFC3339Nano, *in.StartsAt)
	en, err2 := time.Parse(time.RFC3339Nano, *in.EndsAt)
	return err1 == nil && err2 == nil && en.After(st)
}

func (s *Server) createWindow(c *gin.Context) {
	var in windowInput
	if err := c.ShouldBindJSON(&in); err != nil || !validWindow(&in) {
		fail(c, http.StatusBadRequest, "title, startsAt, endsAt (RFC3339, end after start) required")
		return
	}
	ids, _ := json.Marshal(in.MonitorIDs)
	if in.MonitorIDs == nil {
		ids = []byte("[]")
	}
	id := newID("mw")
	if _, err := s.db.Exec(`INSERT INTO maintenance_windows (id, title, starts_at, ends_at, monitor_ids)
		VALUES (?,?,?,?,?)`, id, *in.Title, *in.StartsAt, *in.EndsAt, string(ids)); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	w, err := scanWindow(s.db.QueryRow(`SELECT `+windowCols+` FROM maintenance_windows WHERE id = ?`, id))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, w)
}

func (s *Server) deleteWindow(c *gin.Context) {
	res, err := s.db.Exec(`DELETE FROM maintenance_windows WHERE id = ?`, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "window not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// inMaintenance reports whether a monitor is covered by an active window —
// suppresses auto-incidents and transition alerts.
func (s *Server) inMaintenance(monitorID string) bool {
	rows, err := s.db.Query(`SELECT monitor_ids FROM maintenance_windows
		WHERE datetime(starts_at) <= datetime('now') AND datetime(ends_at) > datetime('now')`)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if rows.Scan(&raw) != nil {
			continue
		}
		var ids []string
		_ = json.Unmarshal([]byte(raw), &ids)
		if len(ids) == 0 {
			return true
		}
		for _, id := range ids {
			if id == monitorID {
				return true
			}
		}
	}
	return false
}

// ---------- status pages ----------

type StatusPage struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	MonitorIDs  []string `json:"monitorIds"`
	SystemIDs   []string `json:"systemIds"`
	CreatedAt   string   `json:"createdAt"`
}

func scanStatusPage(row interface{ Scan(...any) error }) (*StatusPage, error) {
	var p StatusPage
	var mids, sids string
	err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.Description, &mids, &sids, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(mids), &p.MonitorIDs)
	_ = json.Unmarshal([]byte(sids), &p.SystemIDs)
	if p.MonitorIDs == nil {
		p.MonitorIDs = []string{}
	}
	if p.SystemIDs == nil {
		p.SystemIDs = []string{}
	}
	return &p, nil
}

const statusPageCols = `id, slug, title, description, monitor_ids, system_ids, created_at`

func (s *Server) listStatusPages(c *gin.Context) {
	rows, err := s.db.Query(`SELECT ` + statusPageCols + ` FROM status_pages ORDER BY title`)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []StatusPage{}
	for rows.Next() {
		p, err := scanStatusPage(rows)
		if err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, *p)
	}
	c.JSON(http.StatusOK, out)
}

type statusPageInput struct {
	Slug        *string  `json:"slug"`
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	MonitorIDs  []string `json:"monitorIds"`
	SystemIDs   []string `json:"systemIds"`
}

// validSlug keeps the public URL safe: lowercase letters, digits, dashes.
func validSlug(slug string) bool {
	if slug == "" || len(slug) > 64 {
		return false
	}
	for _, r := range slug {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

func (s *Server) createStatusPage(c *gin.Context) {
	var in statusPageInput
	if err := c.ShouldBindJSON(&in); err != nil || in.Title == nil || *in.Title == "" {
		fail(c, http.StatusBadRequest, "title required")
		return
	}
	slug := strOr(in.Slug)
	if slug == "" {
		slug = slugify(*in.Title)
	}
	if !validSlug(slug) {
		fail(c, http.StatusBadRequest, "invalid slug")
		return
	}
	mids, _ := json.Marshal(in.MonitorIDs)
	sids, _ := json.Marshal(in.SystemIDs)
	id := newID("sp")
	_, err := s.db.Exec(`INSERT INTO status_pages (id, slug, title, description, monitor_ids, system_ids)
		VALUES (?,?,?,?,?,?)`, id, slug, *in.Title, strOr(in.Description), string(mids), string(sids))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			fail(c, http.StatusConflict, "slug already in use")
			return
		}
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	p, err := scanStatusPage(s.db.QueryRow(`SELECT `+statusPageCols+` FROM status_pages WHERE id = ?`, id))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (s *Server) patchStatusPage(c *gin.Context) {
	id := c.Param("id")
	if _, err := scanStatusPage(s.db.QueryRow(
		`SELECT `+statusPageCols+` FROM status_pages WHERE id = ?`, id)); notFound(err) {
		fail(c, http.StatusNotFound, "status page not found")
		return
	}
	var in statusPageInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Slug != nil {
		if !validSlug(*in.Slug) {
			fail(c, http.StatusBadRequest, "invalid slug")
			return
		}
		_, _ = s.db.Exec(`UPDATE status_pages SET slug = ? WHERE id = ?`, *in.Slug, id)
	}
	if in.Title != nil && *in.Title != "" {
		_, _ = s.db.Exec(`UPDATE status_pages SET title = ? WHERE id = ?`, *in.Title, id)
	}
	if in.Description != nil {
		_, _ = s.db.Exec(`UPDATE status_pages SET description = ? WHERE id = ?`, *in.Description, id)
	}
	if in.MonitorIDs != nil {
		b, _ := json.Marshal(in.MonitorIDs)
		_, _ = s.db.Exec(`UPDATE status_pages SET monitor_ids = ? WHERE id = ?`, string(b), id)
	}
	if in.SystemIDs != nil {
		b, _ := json.Marshal(in.SystemIDs)
		_, _ = s.db.Exec(`UPDATE status_pages SET system_ids = ? WHERE id = ?`, string(b), id)
	}
	p, err := scanStatusPage(s.db.QueryRow(`SELECT `+statusPageCols+` FROM status_pages WHERE id = ?`, id))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, p)
}

func (s *Server) deleteStatusPage(c *gin.Context) {
	res, err := s.db.Exec(`DELETE FROM status_pages WHERE id = ?`, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "status page not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// ---------- public status ----------

// publicStatus renders one status page's live view: monitors with uptime,
// systems with status, open incidents, active/upcoming maintenance.
func (s *Server) publicStatus(c *gin.Context) {
	p, err := scanStatusPage(s.db.QueryRow(
		`SELECT `+statusPageCols+` FROM status_pages WHERE slug = ?`, c.Param("slug")))
	if notFound(err) {
		fail(c, http.StatusNotFound, "status page not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	pick := func(ids []string) (string, []any) {
		if len(ids) == 0 {
			return "", nil
		}
		marks := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
		args := make([]any, len(ids))
		for i, id := range ids {
			args[i] = id
		}
		return marks, args
	}

	type monStatus struct {
		ID        string  `json:"id"`
		Name      string  `json:"name"`
		Status    string  `json:"status"` // up|down|pending|maintenance
		Uptime24h float64 `json:"uptime24h"`
	}
	mons := []monStatus{}
	mq := `SELECT id, name, status FROM monitors WHERE active = 1`
	marks, args := pick(p.MonitorIDs)
	if marks != "" {
		mq += ` AND id IN (` + marks + `)`
	}
	mq += ` ORDER BY name`
	if rows, err := s.db.Query(mq, args...); err == nil {
		for rows.Next() {
			var m monStatus
			if rows.Scan(&m.ID, &m.Name, &m.Status) == nil {
				if s.inMaintenance(m.ID) {
					m.Status = "maintenance"
				}
				m.Uptime24h = s.monitorStats(m.ID).Uptime24h
				mons = append(mons, m)
			}
		}
		rows.Close()
	}

	type sysStatus struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	systems := []sysStatus{}
	sq := `SELECT id, name, status FROM systems`
	marks, args = pick(p.SystemIDs)
	if marks != "" {
		sq += ` WHERE id IN (` + marks + `)`
	}
	sq += ` ORDER BY name`
	if rows, err := s.db.Query(sq, args...); err == nil {
		for rows.Next() {
			var sm sysStatus
			if rows.Scan(&sm.ID, &sm.Name, &sm.Status) == nil {
				systems = append(systems, sm)
			}
		}
		rows.Close()
	}

	incidents := []Incident{}
	if rows, err := s.db.Query(`SELECT ` + incidentCols + ` FROM incidents
		WHERE status IN ('open','ack','resolved')
		ORDER BY created_at DESC LIMIT 20`); err == nil {
		for rows.Next() {
			in, err := scanIncident(rows)
			if err == nil {
				in.Updates = s.incidentUpdates(in.ID)
				incidents = append(incidents, *in)
			}
		}
		rows.Close()
	}

	maintenance := []MaintWindow{}
	if rows, err := s.db.Query(`SELECT ` + windowCols + ` FROM maintenance_windows
		WHERE datetime(ends_at) > datetime('now') ORDER BY starts_at LIMIT 10`); err == nil {
		for rows.Next() {
			if w, err := scanWindow(rows); err == nil {
				maintenance = append(maintenance, *w)
			}
		}
		rows.Close()
	}

	// Overall state: down wins over maintenance wins over pending.
	overall := "up"
	for _, m := range mons {
		switch m.Status {
		case "down":
			overall = "down"
		case "maintenance":
			if overall == "up" {
				overall = "maintenance"
			}
		case "pending":
			if overall == "up" {
				overall = "pending"
			}
		}
	}
	for _, sm := range systems {
		if sm.Status == "down" {
			overall = "down"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"title":       p.Title,
		"slug":        p.Slug,
		"description": p.Description,
		"overall":     overall,
		"monitors":    mons,
		"systems":     systems,
		"incidents":   incidents,
		"maintenance": maintenance,
	})
}

// ---------- auto-incident hook ----------

// autoIncident opens an incident when a monitor goes down and resolves the
// matching open incident when it recovers. Suppressed during maintenance.
func (s *Server) autoIncident(monitorID, name, next string) {
	if s.inMaintenance(monitorID) {
		return
	}
	if next == "down" {
		// Don't stack duplicates on a flap loop.
		var open string
		if err := s.db.QueryRow(`SELECT id FROM incidents
			WHERE monitor_id = ? AND status IN ('open','ack')`, monitorID).Scan(&open); err == nil {
			return
		}
		id := newID("inc")
		title := fmt.Sprintf("Monitor %q is down", name)
		if _, err := s.db.Exec(`INSERT INTO incidents (id, title, severity, status, monitor_id)
			VALUES (?,?,'major','open',?)`, id, title, monitorID); err != nil {
			s.log.Error("auto incident", zap.Error(err))
			return
		}
		_, _ = s.db.Exec(`INSERT INTO incident_updates (incident_id, status, message)
			VALUES (?,'open','Opened automatically — monitor reported down.')`, id)
		return
	}
	if next == "up" {
		res, err := s.db.Exec(`UPDATE incidents SET status = 'resolved',
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
			WHERE monitor_id = ? AND status IN ('open','ack')`, monitorID)
		if err != nil {
			return
		}
		if n, _ := res.RowsAffected(); n > 0 {
			rows, _ := s.db.Query(`SELECT id FROM incidents WHERE monitor_id = ? AND status = 'resolved'`, monitorID)
			var ids []string
			if rows != nil {
				for rows.Next() {
					var id string
					if rows.Scan(&id) == nil {
						ids = append(ids, id)
					}
				}
				rows.Close()
			}
			for _, id := range ids {
				_, _ = s.db.Exec(`INSERT INTO incident_updates (incident_id, status, message)
					VALUES (?,'resolved','Resolved automatically — monitor recovered.')`, id)
			}
		}
	}
}
