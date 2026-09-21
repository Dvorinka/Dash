package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// /api/metrics — Prometheus text exposition over the tables we already have.
// No registry, no deps: the counts are one-line SQL each.

func promEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(s)
}

func (s *Server) metrics(c *gin.Context) {
	var b strings.Builder
	count := func(q string) int {
		var n int
		_ = s.db.QueryRow(q).Scan(&n)
		return n
	}
	gauge := func(name, help string, v int) {
		fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s gauge\n%s %d\n", name, help, name, name, v)
	}

	gauge("dash_monitors_total", "Configured monitors", count(`SELECT COUNT(*) FROM monitors`))
	gauge("dash_monitors_up", "Monitors currently up", count(`SELECT COUNT(*) FROM monitors WHERE status='up'`))
	gauge("dash_monitors_down", "Monitors currently down", count(`SELECT COUNT(*) FROM monitors WHERE status='down'`))
	gauge("dash_domains_total", "Tracked domains", count(`SELECT COUNT(*) FROM domains`))
	gauge("dash_domains_expired", "Domains past expiry", count(`SELECT COUNT(*) FROM domains WHERE expiry_date IS NOT NULL AND datetime(expiry_date) < datetime('now')`))
	gauge("dash_domains_expiring_30d", "Domains expiring within 30 days",
		count(`SELECT COUNT(*) FROM domains WHERE expiry_date IS NOT NULL
			AND datetime(expiry_date) >= datetime('now')
			AND datetime(expiry_date) <= datetime('now','+30 days')`))
	gauge("dash_systems_total", "Registered systems", count(`SELECT COUNT(*) FROM systems`))
	gauge("dash_systems_up", "Systems currently reporting", count(`SELECT COUNT(*) FROM systems WHERE status='up'`))
	gauge("dash_systems_down", "Systems gone silent", count(`SELECT COUNT(*) FROM systems WHERE status='down'`))
	gauge("dash_incidents_open", "Open or acknowledged incidents", count(`SELECT COUNT(*) FROM incidents WHERE status IN ('open','ack')`))
	gauge("dash_heartbeats_total", "Recorded monitor heartbeats", count(`SELECT COUNT(*) FROM heartbeats`))

	// Per-monitor up/down gauge — label cardinality is user-controlled and small.
	b.WriteString("# TYPE dash_monitor_up gauge\n")
	if rows, err := s.db.Query(`SELECT name, status FROM monitors`); err == nil {
		for rows.Next() {
			var name, st string
			if rows.Scan(&name, &st) == nil {
				v := 0
				if st == "up" {
					v = 1
				}
				fmt.Fprintf(&b, "dash_monitor_up{name=%q} %d\n", promEscape(name), v)
			}
		}
		rows.Close()
	}
	b.WriteString("# TYPE dash_system_up gauge\n")
	if rows, err := s.db.Query(`SELECT name, status FROM systems`); err == nil {
		for rows.Next() {
			var name, st string
			if rows.Scan(&name, &st) == nil {
				v := 0
				if st == "up" {
					v = 1
				}
				fmt.Fprintf(&b, "dash_system_up{name=%q} %d\n", promEscape(name), v)
			}
		}
		rows.Close()
	}

	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(b.String()))
}
