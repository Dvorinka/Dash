package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Export is the portable backup shape: version + settings + board tree.
type Export struct {
	Version  int                        `json:"version"`
	Settings map[string]json.RawMessage `json:"settings"`
	Sections []Section                  `json:"sections"`
}

func (s *Server) exportBoard(c *gin.Context) {
	settings, err := s.allSettings()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	sections, err := s.board()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, Export{Version: 1, Settings: settings, Sections: sections})
}

// importBoard replaces board state inside one transaction. Ids from the
// payload are preserved so a round-trip restores identical state.
func (s *Server) importBoard(c *gin.Context) {
	var in Export
	if err := c.ShouldBindJSON(&in); err != nil || in.Sections == nil {
		fail(c, http.StatusBadRequest, "invalid export payload")
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback() }()

	for _, q := range []string{`DELETE FROM urls`, `DELETE FROM items`, `DELETE FROM sections`} {
		if _, err := tx.Exec(q); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	for _, sec := range in.Sections {
		if sec.ID == "" || sec.Name == "" {
			fail(c, http.StatusBadRequest, "section missing id or name")
			return
		}
		if _, err := tx.Exec(`INSERT INTO sections (id, name, position, collapsed) VALUES (?, ?, ?, ?)`,
			sec.ID, sec.Name, sec.Position, sec.Collapsed); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		for _, it := range sec.Items {
			if it.ID == "" || it.Name == "" {
				fail(c, http.StatusBadRequest, "item missing id or name")
				return
			}
			var cfg any
			if it.Config != nil {
				cfg = string(*it.Config)
			}
			if _, err := tx.Exec(
				`INSERT INTO items (id, section_id, kind, name, icon, position, config) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				it.ID, sec.ID, orDefault(it.Kind, "service"), it.Name, it.Icon, it.Position, cfg); err != nil {
				fail(c, http.StatusInternalServerError, err.Error())
				return
			}
			for _, u := range it.URLs {
				if !validURL(u.URL) {
					fail(c, http.StatusBadRequest, "invalid url: "+u.URL)
					return
				}
				uid := u.ID
				if uid == "" {
					uid = newID("u")
				}
				if _, err := tx.Exec(`INSERT INTO urls (id, item_id, url, label, position) VALUES (?, ?, ?, ?, ?)`,
					uid, it.ID, u.URL, u.Label, u.Position); err != nil {
					fail(c, http.StatusInternalServerError, err.Error())
					return
				}
			}
		}
	}
	for k, v := range in.Settings {
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

	sections, err := s.board()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, sections)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
