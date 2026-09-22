package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdvorak/dash/internal/domain"
	"go.uber.org/zap"
)

// Subdomain discovery: crt.sh + DNS, persisted per domain and swept at most
// once a day inside the domain refresh. CT names are untrusted strings —
// stored as text, rendered as text, never linked.

type subdomainView struct {
	Name      string   `json:"name"`
	IPs       []string `json:"ips"`
	FirstSeen string   `json:"firstSeen"`
	LastSeen  string   `json:"lastSeen"`
}

func (s *Server) listSubdomains(c *gin.Context) {
	rows, err := s.db.Query(`SELECT name, ips, first_seen, last_seen
		FROM subdomains WHERE domain_id = ? ORDER BY name`, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []subdomainView{}
	for rows.Next() {
		var v subdomainView
		var ips string
		if err := rows.Scan(&v.Name, &ips, &v.FirstSeen, &v.LastSeen); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(ips), &v.IPs)
		if v.IPs == nil {
			v.IPs = []string{}
		}
		out = append(out, v)
	}
	c.JSON(http.StatusOK, out)
}

// refreshSubdomains forces a discovery run for one domain.
func (s *Server) refreshSubdomains(c *gin.Context) {
	id := c.Param("id")
	d, err := s.loadDomain(id)
	if notFound(err) {
		fail(c, http.StatusNotFound, "domain not found")
		return
	}
	s.sweepSubdomains(d)
	s.listSubdomains(c)
}

// sweepSubdomains runs CT+DNS discovery and upserts rows. Called from
// refreshDomain when sub_checked_at is stale (>24h), and on manual refresh.
func (s *Server) sweepSubdomains(d *domainRow) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	found, err := domain.DiscoverSubdomains(ctx, d.Name)
	if err != nil {
		s.log.Warn("subdomain discovery", zap.String("domain", d.Name), zap.Error(err))
		return
	}
	for _, f := range found {
		ips, _ := json.Marshal(f.IPs)
		if _, err := s.db.Exec(`INSERT INTO subdomains (id, domain_id, name, ips)
			VALUES (?,?,?,?)
			ON CONFLICT(domain_id, name) DO UPDATE SET ips = excluded.ips,
				last_seen = strftime('%Y-%m-%dT%H:%M:%fZ','now')`,
			newID("sub"), d.ID, f.Name, string(ips)); err != nil {
			s.log.Error("subdomain upsert", zap.Error(err))
		}
	}
	_, _ = s.db.Exec(`UPDATE domains SET sub_checked_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`, d.ID)
}

// maybeSweepSubdomains gates discovery to once per day per domain.
func (s *Server) maybeSweepSubdomains(d *domainRow) {
	var checked *string
	_ = s.db.QueryRow(`SELECT sub_checked_at FROM domains WHERE id = ?`, d.ID).Scan(&checked)
	if checked != nil && *checked != "" {
		if t, err := time.Parse("2006-01-02T15:04:05.000Z", *checked); err == nil &&
			time.Since(t) < 24*time.Hour {
			return
		}
	}
	s.sweepSubdomains(d)
}
