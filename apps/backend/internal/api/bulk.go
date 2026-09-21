package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// POST /api/import/csv?kind=monitors|domains — header-driven CSV bulk import.
// Monitors: name,type,url,hostname,port,interval — name + (url|hostname) needed.
// Domains:  name — one per row.

func (s *Server) importCSV(c *gin.Context) {
	kind := c.DefaultQuery("kind", "monitors")
	if kind != "monitors" && kind != "domains" {
		fail(c, http.StatusBadRequest, "kind must be monitors or domains")
		return
	}
	rd := csv.NewReader(c.Request.Body)
	rd.FieldsPerRecord = -1 // tolerate ragged rows; header drives positions
	recs, err := rd.ReadAll()
	if err != nil || len(recs) < 2 {
		fail(c, http.StatusBadRequest, "csv with a header row and at least one entry required")
		return
	}
	head := map[string]int{}
	for i, h := range recs[0] {
		head[strings.ToLower(strings.TrimSpace(h))] = i
	}
	cell := func(rec []string, col string) string {
		if i, ok := head[col]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}

	var created, skipped int
	var errs []string
	for i, rec := range recs[1:] {
		row := i + 2 // header is row 1
		name := cell(rec, "name")
		if name == "" {
			skipped++
			continue
		}
		if kind == "domains" {
			if _, err := s.db.Exec(`INSERT INTO domains (id, name) VALUES (?, ?)`,
				newID("d"), strings.ToLower(name)); err != nil {
				if strings.Contains(err.Error(), "UNIQUE") {
					skipped++
					continue
				}
				errs = append(errs, fmt.Sprintf("row %d: %v", row, err))
				continue
			}
			created++
			continue
		}

		// monitors
		typ := orDefault(cell(rec, "type"), "http")
		if !validMonitorType(typ) {
			errs = append(errs, fmt.Sprintf("row %d: bad type %q", row, typ))
			continue
		}
		url, host := cell(rec, "url"), cell(rec, "hostname")
		if url == "" && host == "" {
			errs = append(errs, fmt.Sprintf("row %d: url or hostname required", row))
			continue
		}
		port, _ := strconv.Atoi(cell(rec, "port"))
		interval, _ := strconv.Atoi(cell(rec, "interval"))
		if interval <= 0 {
			interval = 60
		}
		token := ""
		if typ == "push" {
			token = newID("push")
		}
		if _, err := s.db.Exec(`INSERT INTO monitors
			(id, name, type, url, hostname, port, interval_s, push_token, position)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			newID("m"), name, typ, url, host, port, interval, token,
			float64(time.Now().UnixNano())/1e9); err != nil {
			errs = append(errs, fmt.Sprintf("row %d: %v", row, err))
			continue
		}
		created++
	}
	c.JSON(http.StatusOK, gin.H{"created": created, "skipped": skipped, "errors": errs})
}
