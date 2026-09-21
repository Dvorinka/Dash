package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdvorak/dash/internal/monitor"
	"github.com/tdvorak/dash/internal/widget"
	"go.uber.org/zap"
)

// ---------- model ----------

const monitorCols = `id, name, type, url, hostname, port, method, headers, body,
	keyword, keyword_invert, json_query, expected, dns_type, interval_s, timeout_s,
	retries, active, status, push_token, tags, notes, position, last_check, created_at`

func scanMonitor(row interface{ Scan(...any) error }) (*monitor.Monitor, error) {
	var m monitor.Monitor
	var tags string
	var invert, active int
	err := row.Scan(&m.ID, &m.Name, &m.Type, &m.URL, &m.Hostname, &m.Port, &m.Method,
		&m.Headers, &m.Body, &m.Keyword, &invert, &m.JSONQuery, &m.Expected, &m.DNSType,
		&m.IntervalS, &m.TimeoutS, &m.Retries, &active, &m.Status, &m.PushToken,
		&tags, &m.Notes, &m.Position, &m.LastCheck, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	m.KeywordInvert = invert != 0
	m.Active = active != 0
	_ = json.Unmarshal([]byte(tags), &m.Tags)
	if m.Tags == nil {
		m.Tags = []string{}
	}
	return &m, nil
}

func (s *Server) loadMonitor(id string) (*monitor.Monitor, error) {
	row := s.db.QueryRow(`SELECT `+monitorCols+` FROM monitors WHERE id = ?`, id)
	return scanMonitor(row)
}

func (s *Server) loadMonitorByToken(token string) (*monitor.Monitor, error) {
	row := s.db.QueryRow(`SELECT `+monitorCols+` FROM monitors WHERE push_token = ? AND push_token != ''`, token)
	return scanMonitor(row)
}

func (s *Server) listMonitors() ([]monitor.Monitor, error) {
	rows, err := s.db.Query(`SELECT ` + monitorCols + ` FROM monitors ORDER BY position, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []monitor.Monitor
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// ---------- uptime stats ----------

type uptimeStats struct {
	Up         int     `json:"up"`
	Down       int     `json:"down"`
	Uptime24h  float64 `json:"uptime24h"`
	Uptime30d  float64 `json:"uptime30d"`
	AvgPing24h int     `json:"avgPing24h"`
}

func (s *Server) monitorStats(id string) uptimeStats {
	var st uptimeStats
	pct := func(hours int) float64 {
		var up, down int
		_ = s.db.QueryRow(`SELECT COALESCE(SUM(status='up'),0), COALESCE(SUM(status='down'),0)
			FROM heartbeats WHERE monitor_id = ? AND checked_at >= datetime('now', ?)`,
			id, "-"+strconv.Itoa(hours)+" hours").Scan(&up, &down)
		if up+down == 0 {
			return 100
		}
		return float64(up) / float64(up+down) * 100
	}
	st.Uptime24h = pct(24)
	st.Uptime30d = pct(720)
	_ = s.db.QueryRow(`SELECT COALESCE(AVG(NULLIF(ping_ms,0)),0) FROM heartbeats
		WHERE monitor_id = ? AND checked_at >= datetime('now','-24 hours')`, id).Scan(&st.AvgPing24h)
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(status='up'),0), COALESCE(SUM(status='down'),0)
		FROM heartbeats WHERE monitor_id = ?`, id).Scan(&st.Up, &st.Down)
	return st
}

// monitorView is the API shape: the row plus live uptime stats.
type monitorView struct {
	*monitor.Monitor
	Stats uptimeStats `json:"stats"`
}

// ---------- handlers ----------

func (s *Server) listMonitorsH(c *gin.Context) {
	ms, err := s.listMonitors()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]monitorView, 0, len(ms))
	for i := range ms {
		out = append(out, monitorView{Monitor: &ms[i], Stats: s.monitorStats(ms[i].ID)})
	}
	c.JSON(http.StatusOK, out)
}

// monitorInput is the create/update payload — all fields optional on PATCH.
type monitorInput struct {
	Name          *string  `json:"name"`
	Type          *string  `json:"type"`
	URL           *string  `json:"url"`
	Hostname      *string  `json:"hostname"`
	Port          *int     `json:"port"`
	Method        *string  `json:"method"`
	Headers       *string  `json:"headers"`
	Body          *string  `json:"body"`
	Keyword       *string  `json:"keyword"`
	KeywordInvert *bool    `json:"keywordInvert"`
	JSONQuery     *string  `json:"jsonQuery"`
	Expected      *string  `json:"expected"`
	DNSType       *string  `json:"dnsType"`
	IntervalS     *int     `json:"intervalS"`
	TimeoutS      *int     `json:"timeoutS"`
	Retries       *int     `json:"retries"`
	Active        *bool    `json:"active"`
	Tags          []string `json:"tags"`
	Notes         *string  `json:"notes"`
}

func validMonitorType(t string) bool {
	for _, v := range monitor.Types() {
		if v == t {
			return true
		}
	}
	return false
}

func strOr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func strOr2(p *string, def string) string {
	if p == nil || *p == "" {
		return def
	}
	return *p
}

func intOr(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func intOr2(p *int, def int) int {
	if p == nil || *p <= 0 {
		return def
	}
	return *p
}

func boolOr(p *bool) bool { return p != nil && *p }

func boolOr2(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func (s *Server) createMonitor(c *gin.Context) {
	var in monitorInput
	if err := c.ShouldBindJSON(&in); err != nil || in.Name == nil || in.Type == nil {
		fail(c, http.StatusBadRequest, "name and type required")
		return
	}
	if !validMonitorType(*in.Type) {
		fail(c, http.StatusBadRequest, "invalid type")
		return
	}
	id := newID("m")
	token := ""
	if *in.Type == "push" {
		token = newID("push")
	}
	tags, _ := json.Marshal(in.Tags)
	if in.Tags == nil {
		tags = []byte("[]")
	}
	_, err := s.db.Exec(`INSERT INTO monitors
		(id, name, type, url, hostname, port, method, headers, body, keyword,
		 keyword_invert, json_query, expected, dns_type, interval_s, timeout_s,
		 retries, active, push_token, tags, notes, position)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, *in.Name, *in.Type, strOr(in.URL), strOr(in.Hostname), intOr(in.Port),
		strOr2(in.Method, "GET"), strOr2(in.Headers, "{}"), strOr(in.Body),
		strOr(in.Keyword), boolOr(in.KeywordInvert), strOr(in.JSONQuery),
		strOr(in.Expected), strOr2(in.DNSType, "A"), intOr2(in.IntervalS, 60),
		intOr2(in.TimeoutS, 10), intOr(in.Retries), boolOr2(in.Active, true),
		token, string(tags), strOr(in.Notes), float64(time.Now().UnixNano())/1e9)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	m, err := s.loadMonitor(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, monitorView{Monitor: m, Stats: s.monitorStats(id)})
}

func (s *Server) getMonitor(c *gin.Context) {
	m, err := s.loadMonitor(c.Param("id"))
	if notFound(err) {
		fail(c, http.StatusNotFound, "monitor not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, monitorView{Monitor: m, Stats: s.monitorStats(m.ID)})
}

// patchMonitor applies a sparse update; `active` flips pause/resume.
func (s *Server) patchMonitor(c *gin.Context) {
	id := c.Param("id")
	if _, err := s.loadMonitor(id); notFound(err) {
		fail(c, http.StatusNotFound, "monitor not found")
		return
	}
	var in monitorInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Type != nil && !validMonitorType(*in.Type) {
		fail(c, http.StatusBadRequest, "invalid type")
		return
	}
	setStr := func(col string, p *string) {
		if p != nil {
			_, _ = s.db.Exec(`UPDATE monitors SET `+col+` = ? WHERE id = ?`, *p, id)
		}
	}
	setStr("name", in.Name)
	setStr("type", in.Type)
	setStr("url", in.URL)
	setStr("hostname", in.Hostname)
	setStr("method", in.Method)
	setStr("headers", in.Headers)
	setStr("body", in.Body)
	setStr("keyword", in.Keyword)
	setStr("json_query", in.JSONQuery)
	setStr("expected", in.Expected)
	setStr("dns_type", in.DNSType)
	setStr("notes", in.Notes)
	setInt := func(col string, p *int) {
		if p != nil {
			_, _ = s.db.Exec(`UPDATE monitors SET `+col+` = ? WHERE id = ?`, *p, id)
		}
	}
	setInt("port", in.Port)
	setInt("interval_s", in.IntervalS)
	setInt("timeout_s", in.TimeoutS)
	setInt("retries", in.Retries)
	if in.KeywordInvert != nil {
		_, _ = s.db.Exec(`UPDATE monitors SET keyword_invert = ? WHERE id = ?`, *in.KeywordInvert, id)
	}
	if in.Tags != nil {
		tags, _ := json.Marshal(in.Tags)
		_, _ = s.db.Exec(`UPDATE monitors SET tags = ? WHERE id = ?`, string(tags), id)
	}
	if in.Active != nil {
		// pause/resume flips the stored status back to pending for a fresh probe
		status := "pending"
		if !*in.Active {
			status = "paused"
		}
		_, _ = s.db.Exec(`UPDATE monitors SET active = ?, status = ? WHERE id = ?`, *in.Active, status, id)
	}
	m, err := s.loadMonitor(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, monitorView{Monitor: m, Stats: s.monitorStats(id)})
}

func (s *Server) deleteMonitor(c *gin.Context) {
	res, err := s.db.Exec(`DELETE FROM monitors WHERE id = ?`, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "monitor not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// heartbeats returns the recent check history for charts/logs.
func (s *Server) monitorHeartbeats(c *gin.Context) {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	if hours <= 0 || hours > 24*30 {
		hours = 24
	}
	rows, err := s.db.Query(`SELECT id, status, ping_ms, msg, cert_expiry, checked_at
		FROM heartbeats WHERE monitor_id = ? AND checked_at >= datetime('now', ?)
		ORDER BY checked_at`, c.Param("id"), "-"+strconv.Itoa(hours)+" hours")
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	type hb struct {
		ID         int    `json:"id"`
		Status     string `json:"status"`
		PingMs     int    `json:"pingMs"`
		Msg        string `json:"msg"`
		CertExpiry int    `json:"certExpiry"`
		CheckedAt  string `json:"checkedAt"`
	}
	out := []hb{}
	for rows.Next() {
		var h hb
		if err := rows.Scan(&h.ID, &h.Status, &h.PingMs, &h.Msg, &h.CertExpiry, &h.CheckedAt); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, h)
	}
	c.JSON(http.StatusOK, out)
}

// checkNow runs one check synchronously and returns the refreshed view.
func (s *Server) checkNow(c *gin.Context) {
	m, err := s.loadMonitor(c.Param("id"))
	if notFound(err) {
		fail(c, http.StatusNotFound, "monitor not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	s.runCheck(m)
	m, err = s.loadMonitor(m.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, monitorView{Monitor: m, Stats: s.monitorStats(m.ID)})
}

// pushIngest is the push-monitor endpoint: any hit records an up heartbeat.
// ?msg= and ?ping= are optional annotations.
func (s *Server) pushIngest(c *gin.Context) {
	m, err := s.loadMonitorByToken(c.Param("token"))
	if err != nil {
		fail(c, http.StatusNotFound, "unknown push token")
		return
	}
	msg := c.Query("msg")
	ping, _ := strconv.Atoi(c.Query("ping"))
	s.writeHeartbeat(m, &monitor.Result{Up: true, PingMs: ping, Msg: msg, CertExpiry: -1})
	_, _ = s.db.Exec(`UPDATE monitors SET status = 'up', last_check = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`, m.ID)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ---------- scheduler ----------

// runScheduler sweeps for due monitors every tick; each check runs in a
// bounded worker. Push monitors go stale -> down when silent too long.
func (s *Server) runScheduler(done <-chan struct{}) {
	tick := time.NewTicker(10 * time.Second)
	prune := time.NewTicker(time.Hour)
	defer tick.Stop()
	defer prune.Stop()
	for {
		select {
		case <-done:
			return
		case <-tick.C:
			s.sweep()
		case <-prune.C:
			if _, err := s.db.Exec(`DELETE FROM heartbeats WHERE checked_at < datetime('now','-30 days')`); err != nil {
				s.log.Error("heartbeat prune", zap.Error(err))
			}
		}
	}
}

var checkSem = make(chan struct{}, 8)

func (s *Server) sweep() {
	rows, err := s.db.Query(`SELECT ` + monitorCols + ` FROM monitors
		WHERE active = 1 AND type != 'push'
		AND (last_check IS NULL OR datetime(last_check, '+' || interval_s || ' seconds') <= datetime('now'))`)
	if err != nil {
		s.log.Error("monitor sweep", zap.Error(err))
		return
	}
	var due []monitor.Monitor
	for rows.Next() {
		if m, err := scanMonitor(rows); err == nil {
			due = append(due, *m)
		}
	}
	rows.Close()

	for i := range due {
		m := &due[i]
		checkSem <- struct{}{}
		go func() {
			defer func() { <-checkSem }()
			s.runCheck(m)
		}()
	}

	// Push monitors: mark down when no push arrived within 2x interval.
	if _, err := s.db.Exec(`UPDATE monitors SET status = 'down',
		last_check = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE type = 'push' AND active = 1
		AND last_check IS NOT NULL
		AND datetime(last_check, '+' || (2 * interval_s) || ' seconds') <= datetime('now')
		AND status != 'down'`); err != nil {
		s.log.Error("push stale sweep", zap.Error(err))
	}
}

// runCheck executes one monitor check (with retries) and persists the result.
func (s *Server) runCheck(m *monitor.Monitor) *monitor.Result {
	var res *monitor.Result
	for attempt := 0; attempt <= m.Retries; attempt++ {
		res = monitor.Check(context.Background(), m)
		if res.Up || attempt == m.Retries {
			break
		}
		time.Sleep(2 * time.Second)
	}
	s.writeHeartbeat(m, res)
	prev := m.Status
	next := "down"
	if res.Up {
		next = "up"
	}
	if _, err := s.db.Exec(`UPDATE monitors SET status = ?, last_check = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`,
		next, m.ID); err != nil {
		s.log.Error("monitor update", zap.Error(err))
	}
	if prev != next && prev != "pending" {
		s.onMonitorTransition(m, prev, next)
	}
	return res
}

func (s *Server) writeHeartbeat(m *monitor.Monitor, res *monitor.Result) {
	status := "down"
	if res.Up {
		status = "up"
	}
	if _, err := s.db.Exec(`INSERT INTO heartbeats (monitor_id, status, ping_ms, msg, cert_expiry)
		VALUES (?,?,?,?,?)`, m.ID, status, res.PingMs, truncate(res.Msg, 300), res.CertExpiry); err != nil {
		s.log.Error("heartbeat write", zap.Error(err))
	}
}

// onMonitorTransition is the alert seam — M2 wires notifications here.
func (s *Server) onMonitorTransition(m *monitor.Monitor, prev, next string) {
	s.log.Info("monitor transition",
		zap.String("monitor", m.Name), zap.String("from", prev), zap.String("to", next))
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// ---------- widget fetcher ----------

// monitorWidget serves board tiles bound to a monitor via config.monitorId.
type monitorWidget struct{ db *sql.DB }

func (w *monitorWidget) Meta() widget.Type {
	return widget.Type{
		Type: "monitor", Name: "Monitor",
		Description: "Live status of an uptime monitor",
		Fields: []widget.Field{
			{Key: "monitorId", Label: "Monitor", Required: true},
		},
	}
}

func (w *monitorWidget) Fetch(_ context.Context, cfg json.RawMessage) (any, error) {
	var c struct {
		MonitorID string `json:"monitorId"`
	}
	if err := json.Unmarshal(cfg, &c); err != nil || c.MonitorID == "" {
		return nil, errors.New("config.monitorId required")
	}
	row := w.db.QueryRow(`SELECT `+monitorCols+` FROM monitors WHERE id = ?`, c.MonitorID)
	m, err := scanMonitor(row)
	if err != nil {
		return nil, errors.New("monitor not found")
	}
	var up, down int
	var avgPing float64
	_ = w.db.QueryRow(`SELECT COALESCE(SUM(status='up'),0), COALESCE(SUM(status='down'),0),
		COALESCE(AVG(NULLIF(ping_ms,0)),0)
		FROM heartbeats WHERE monitor_id = ? AND checked_at >= datetime('now','-24 hours')`,
		m.ID).Scan(&up, &down, &avgPing)
	uptime := 100.0
	if up+down > 0 {
		uptime = float64(up) / float64(up+down) * 100
	}
	return map[string]any{
		"name":     m.Name,
		"status":   m.Status,
		"uptime24": uptime,
		"ping":     int(avgPing),
		"type":     m.Type,
	}, nil
}
