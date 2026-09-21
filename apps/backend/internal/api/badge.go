package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Stateless SVG shields rendered from live rows: /api/badge/<kind>/<id>.svg
// kinds: monitor (status+uptime), domain (days to expiry), system (status+cpu).

var badgeColors = map[string]string{
	"up":       "#3fb950",
	"down":     "#f85149",
	"pending":  "#8b949e",
	"expired":  "#f85149",
	"expiring": "#d29922",
	"ok":       "#3fb950",
	"unknown":  "#8b949e",
}

func badgeSVG(c *gin.Context, label, value, color string) {
	// Fixed-width font assumption is deliberate — shields embed anywhere.
	lw := 6*len(label) + 12
	vw := 6*len(value) + 12
	c.Header("Content-Type", "image/svg+xml")
	c.Header("Cache-Control", "no-cache")
	c.String(http.StatusOK, fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">
<linearGradient id="s" x2="0" y2="100%%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset="1" stop-opacity=".7"/></linearGradient>
<clipPath id="r"><rect width="%d" height="20" rx="3"/></clipPath>
<g clip-path="url(#r)">
<rect width="%d" height="20" fill="#3a3f46"/>
<rect x="%d" width="%d" height="20" fill="%s"/>
<rect width="%d" height="20" fill="url(#s)" fill-opacity=".1"/>
</g>
<g fill="#fff" font-family="DejaVu Sans,Verdana,sans-serif" font-size="11">
<text x="%d" y="14">%s</text>
<text x="%d" y="14">%s</text>
</g></svg>`,
		lw+vw, label, value, lw+vw, lw, lw, vw, color, lw+vw, 6, label, lw+6, value))
}

func (s *Server) badge(c *gin.Context) {
	kind := c.Param("kind")
	// Route is /badge/<kind>/<file>; the conventional suffix is optional.
	id := strings.TrimSuffix(c.Param("file"), ".svg")

	switch kind {
	case "monitor":
		var name, status string
		err := s.db.QueryRow(`SELECT name, status FROM monitors WHERE id = ?`, id).Scan(&name, &status)
		if err != nil {
			badgeSVG(c, "monitor", "unknown", badgeColors["unknown"])
			return
		}
		uptime := s.monitorStats(id).Uptime24h
		badgeSVG(c, name, fmt.Sprintf("%s · %.1f%%", status, uptime), badgeColors[status])
	case "domain":
		var name string
		var expiry *string
		err := s.db.QueryRow(`SELECT name, expiry_date FROM domains WHERE id = ?`, id).Scan(&name, &expiry)
		if err != nil {
			badgeSVG(c, "domain", "unknown", badgeColors["unknown"])
			return
		}
		label, color := "unknown", badgeColors["unknown"]
		if days := daysUntil(expiry); days != nil {
			switch {
			case *days < 0:
				label, color = "expired", badgeColors["expired"]
			case *days <= 30:
				label, color = strconv.Itoa(*days)+"d left", badgeColors["expiring"]
			default:
				label, color = strconv.Itoa(*days)+"d left", badgeColors["ok"]
			}
		}
		badgeSVG(c, name, label, color)
	case "system":
		var name, status string
		err := s.db.QueryRow(`SELECT name, status FROM systems WHERE id = ?`, id).Scan(&name, &status)
		if err != nil {
			badgeSVG(c, "system", "unknown", badgeColors["unknown"])
			return
		}
		badgeSVG(c, name, status, badgeColors[status])
	default:
		fail(c, http.StatusNotFound, "unknown badge kind")
	}
}
