package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// allSettings reads the whole settings table into a key/value map.
func (s *Server) allSettings() (map[string]json.RawMessage, error) {
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = json.RawMessage(v)
	}
	return out, rows.Err()
}

func (s *Server) getSettings(c *gin.Context) {
	m, err := s.allSettings()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, m)
}

// putSettings merges the posted keys into the settings table.
func (s *Server) putSettings(c *gin.Context) {
	var in map[string]json.RawMessage
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback() }()
	for k, v := range in {
		if _, err := tx.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, string(v)); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if err := tx.Commit(); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	m, err := s.allSettings()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, m)
}
