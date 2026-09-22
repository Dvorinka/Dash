package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tdvorak/dash/internal/widget"
	"go.uber.org/zap"
)

// ---------- model ----------

// StatSample is one agent payload. Numeric fields are optional-ish: the
// agent may omit what a platform cannot provide.
type StatSample struct {
	Host       string            `json:"host"`
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	CPUModel   string            `json:"cpuModel"`
	Cores      int               `json:"cores"`
	IntervalS  int               `json:"intervalS"`
	UptimeS    float64           `json:"uptimeS"`
	CPU        float64           `json:"cpu"`      // percent
	MemTotal   float64           `json:"memTotal"` // bytes
	MemUsed    float64           `json:"memUsed"`
	SwapTotal  float64           `json:"swapTotal"`
	SwapUsed   float64           `json:"swapUsed"`
	DiskTotal  float64           `json:"diskTotal"` // bytes, root fs
	DiskUsed   float64           `json:"diskUsed"`
	NetRx      float64           `json:"netRx"` // bytes/sec
	NetTx      float64           `json:"netTx"`
	Load1      float64           `json:"load1"`
	Load5      float64           `json:"load5"`
	Load15     float64           `json:"load15"`
	Temps      map[string]float64 `json:"temps"`
	Containers []ContainerStat   `json:"containers"`
	// Optional collectors — present only when the host has the tools.
	Smart []SmartDisk `json:"smart,omitempty"`
	ZFS   []ZFSPool   `json:"zfs,omitempty"`
	GPU   []GPUStat   `json:"gpu,omitempty"`
}

// ContainerStat is one Docker container's liveness plus resource use.
// CPU is percent of one core (may exceed 100); Mem* are bytes. Zero when the
// agent couldn't reach the stats endpoint.
type ContainerStat struct {
	Name     string  `json:"name"`
	State    string  `json:"state"`
	CPU      float64 `json:"cpu,omitempty"`
	MemUsed  float64 `json:"memUsed,omitempty"`
	MemLimit float64 `json:"memLimit,omitempty"`
}

// SmartDisk is one drive's SMART summary (smartctl).
type SmartDisk struct {
	Device string  `json:"device"`
	Model  string  `json:"model"`
	Passed *bool   `json:"passed,omitempty"`
	TempC  float64 `json:"tempC,omitempty"`
}

// ZFSPool is one pool's health and space (zpool).
type ZFSPool struct {
	Name   string  `json:"name"`
	Health string  `json:"health"`
	Size   float64 `json:"size"`
	Free   float64 `json:"free"`
}

// GPUStat is one GPU's basic metrics (sysfs or nvidia-smi).
type GPUStat struct {
	Name     string  `json:"name"`
	TempC    float64 `json:"tempC,omitempty"`
	UtilPct  float64 `json:"utilPct,omitempty"`
	MemUsed  float64 `json:"memUsed,omitempty"`
	MemTotal float64 `json:"memTotal,omitempty"`
}

// System is one monitored host; `latest` is the newest StatSample.
type System struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Token      string      `json:"token"`
	Host       string      `json:"host"`
	OS         string      `json:"os"`
	Arch       string      `json:"arch"`
	CPUModel   string      `json:"cpuModel"`
	Cores      int         `json:"cores"`
	IntervalS  int         `json:"intervalS"`
	Status     string      `json:"status"` // pending | up | down
	LastSeenAt *string     `json:"lastSeenAt"`
	Latest     *StatSample `json:"latest"`
	CreatedAt  string      `json:"createdAt"`
}

const systemCols = `id, name, token, host, os, arch, cpu_model, cores,
	interval_s, status, last_seen_at, latest, created_at`

func scanSystem(row interface{ Scan(...any) error }) (*System, error) {
	var sys System
	var latest string
	err := row.Scan(&sys.ID, &sys.Name, &sys.Token, &sys.Host, &sys.OS,
		&sys.Arch, &sys.CPUModel, &sys.Cores, &sys.IntervalS, &sys.Status,
		&sys.LastSeenAt, &latest, &sys.CreatedAt)
	if err != nil {
		return nil, err
	}
	if latest != "" && latest != "{}" {
		var s StatSample
		if json.Unmarshal([]byte(latest), &s) == nil {
			sys.Latest = &s
		}
	}
	return &sys, nil
}

func (s *Server) loadSystem(id string) (*System, error) {
	return scanSystem(s.db.QueryRow(`SELECT `+systemCols+` FROM systems WHERE id = ?`, id))
}

func (s *Server) loadSystemByToken(token string) (*System, error) {
	return scanSystem(s.db.QueryRow(
		`SELECT `+systemCols+` FROM systems WHERE token = ?`, token))
}

// ---------- handlers ----------

func (s *Server) listSystemsH(c *gin.Context) {
	rows, err := s.db.Query(`SELECT ` + systemCols + ` FROM systems ORDER BY name`)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []System{}
	for rows.Next() {
		sys, err := scanSystem(rows)
		if err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, *sys)
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) createSystem(c *gin.Context) {
	var in struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Name == "" {
		fail(c, http.StatusBadRequest, "name required")
		return
	}
	id := newID("sys")
	if _, err := s.db.Exec(`INSERT INTO systems (id, name, token) VALUES (?,?,?)`,
		id, in.Name, newID("agent")); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	sys, err := s.loadSystem(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, sys)
}

func (s *Server) getSystem(c *gin.Context) {
	sys, err := s.loadSystem(c.Param("id"))
	if notFound(err) {
		fail(c, http.StatusNotFound, "system not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, sys)
}

func (s *Server) patchSystem(c *gin.Context) {
	id := c.Param("id")
	if _, err := s.loadSystem(id); notFound(err) {
		fail(c, http.StatusNotFound, "system not found")
		return
	}
	var in struct {
		Name   *string `json:"name"`
		Rotate *bool   `json:"rotateToken"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Name != nil && *in.Name != "" {
		_, _ = s.db.Exec(`UPDATE systems SET name = ? WHERE id = ?`, *in.Name, id)
	}
	if in.Rotate != nil && *in.Rotate {
		_, _ = s.db.Exec(`UPDATE systems SET token = ? WHERE id = ?`, newID("agent"), id)
	}
	sys, err := s.loadSystem(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, sys)
}

func (s *Server) deleteSystem(c *gin.Context) {
	res, err := s.db.Exec(`DELETE FROM systems WHERE id = ?`, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "system not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// systemStats returns stored samples newest-last for charting.
func (s *Server) systemStats(c *gin.Context) {
	if _, err := s.loadSystem(c.Param("id")); notFound(err) {
		fail(c, http.StatusNotFound, "system not found")
		return
	}
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	if hours <= 0 || hours > 24*7 {
		hours = 24
	}
	rows, err := s.db.Query(`SELECT ts, payload FROM system_stats
		WHERE system_id = ? AND ts >= datetime('now', ?) ORDER BY ts`,
		c.Param("id"), "-"+strconv.Itoa(hours)+" hours")
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	type sample struct {
		TS string `json:"ts"`
		StatSample
	}
	out := []sample{}
	for rows.Next() {
		var ts, payload string
		if err := rows.Scan(&ts, &payload); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		var sm sample
		sm.TS = ts
		if json.Unmarshal([]byte(payload), &sm.StatSample) == nil {
			out = append(out, sm)
		}
	}
	c.JSON(http.StatusOK, out)
}

// ingestSystem receives one agent push. Auth: Authorization: Bearer <token>.
func (s *Server) ingestSystem(c *gin.Context) {
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if token == "" {
		token = c.GetHeader("X-Dash-Token")
	}
	if token == "" {
		fail(c, http.StatusUnauthorized, "token required")
		return
	}
	sys, err := s.loadSystemByToken(token)
	if err != nil {
		fail(c, http.StatusUnauthorized, "unknown token")
		return
	}
	var sample StatSample
	if err := c.ShouldBindJSON(&sample); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	if sample.IntervalS <= 0 {
		sample.IntervalS = 10
	}
	payload, _ := json.Marshal(&sample)
	prev := sys.Status
	_, err = s.db.Exec(`UPDATE systems SET
		host = ?, os = ?, arch = ?, cpu_model = ?, cores = ?, interval_s = ?,
		status = 'up', last_seen_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'),
		latest = ?
		WHERE id = ?`,
		sample.Host, sample.OS, sample.Arch, sample.CPUModel, sample.Cores,
		sample.IntervalS, string(payload), sys.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.db.Exec(`INSERT INTO system_stats (system_id, payload) VALUES (?,?)`,
		sys.ID, string(payload)); err != nil {
		s.log.Error("system stats insert", zap.Error(err))
	}
	if prev != "up" {
		s.notify("system.up", sys.Name,
			fmt.Sprintf("System %q is back online (%s)", sys.Name, sample.Host))
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// sweepSystems marks agents down when silent for > 3 intervals (min 30s).
func (s *Server) sweepSystems() {
	rows, err := s.db.Query(`UPDATE systems SET status = 'down'
		WHERE status = 'up' AND last_seen_at IS NOT NULL
		AND datetime(last_seen_at, '+' || MAX(3 * interval_s, 30) || ' seconds') <= datetime('now')
		RETURNING id, name, host`)
	if err != nil {
		s.log.Error("system sweep", zap.Error(err))
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, host string
		if rows.Scan(&id, &name, &host) == nil {
			s.notify("system.down", name,
				fmt.Sprintf("System %q stopped reporting (%s)", name, host))
		}
	}
}

// ---------- widget fetcher ----------

// systemWidget serves board tiles bound to a system via config.systemId.
type systemWidget struct{ db *sql.DB }

func (w *systemWidget) Meta() widget.Type {
	return widget.Type{
		Type: "system", Name: "System",
		Description: "Live CPU/memory/disk of a monitored host",
		Fields: []widget.Field{
			{Key: "systemId", Label: "System", Required: true},
		},
	}
}

func (w *systemWidget) Fetch(_ context.Context, cfg json.RawMessage) (any, error) {
	var c struct {
		SystemID string `json:"systemId"`
	}
	if err := json.Unmarshal(cfg, &c); err != nil || c.SystemID == "" {
		return nil, errors.New("config.systemId required")
	}
	var name, status, latest string
	var lastSeen *string
	err := w.db.QueryRow(`SELECT name, status, last_seen_at, latest
		FROM systems WHERE id = ?`, c.SystemID).Scan(&name, &status, &lastSeen, &latest)
	if err != nil {
		return nil, errors.New("system not found")
	}
	var sm StatSample
	_ = json.Unmarshal([]byte(latest), &sm)
	return map[string]any{
		"name":     name,
		"status":   status,
		"lastSeen": lastSeen,
		"cpu":      sm.CPU,
		"memTotal": sm.MemTotal,
		"memUsed":  sm.MemUsed,
		"diskTotal": sm.DiskTotal,
		"diskUsed":  sm.DiskUsed,
	}, nil
}
