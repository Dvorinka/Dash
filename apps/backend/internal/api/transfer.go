package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// errBadInput marks payload validation failures so import handlers can map
// them to 400 while genuine DB failures stay 500.
var errBadInput = errors.New("invalid import payload")

// failImport maps replaceBoard errors: bad input -> 400, the rest -> 500.
func failImport(c *gin.Context, err error) {
	if errors.Is(err, errBadInput) {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	fail(c, http.StatusInternalServerError, err.Error())
}

// Export is the portable backup shape: version + settings + board tree.
// v2 adds monitoring entities alongside — additive, so v1 files still import.
type Export struct {
	Version  int                        `json:"version"`
	Settings map[string]json.RawMessage `json:"settings"`
	Sections []Section                  `json:"sections"`
	Monitors []json.RawMessage          `json:"monitors,omitempty"`
	Domains  []json.RawMessage          `json:"domains,omitempty"`
	Systems  []json.RawMessage          `json:"systems,omitempty"`
}

// dumpRows serializes each row of a table to JSON via the driver's column
// metadata — schema drift can't silently drop columns from the backup.
func (s *Server) dumpRows(table string) ([]json.RawMessage, error) {
	rows, err := s.db.Query(`SELECT * FROM ` + table) // table is a fixed literal
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := []json.RawMessage{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		obj := map[string]any{}
		for i, col := range cols {
			obj[col] = vals[i]
		}
		b, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
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
	ex := Export{Version: 2, Settings: settings, Sections: sections}
	for _, t := range []struct {
		name string
		dst  *[]json.RawMessage
	}{
		{"monitors", &ex.Monitors},
		{"domains", &ex.Domains},
		{"systems", &ex.Systems},
	} {
		if rows, err := s.dumpRows(t.name); err == nil {
			*t.dst = rows
		}
	}
	c.JSON(http.StatusOK, ex)
}

// importBoard replaces board state inside one transaction. Ids from the
// payload are preserved so a round-trip restores identical state.
func (s *Server) importBoard(c *gin.Context) {
	var in Export
	if err := c.ShouldBindJSON(&in); err != nil || in.Sections == nil {
		fail(c, http.StatusBadRequest, "invalid export payload")
		return
	}
	if err := s.replaceBoard(in.Sections, in.Settings); err != nil {
		failImport(c, err)
		return
	}
	sections, err := s.board()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, sections)
}

// replaceBoard swaps all board rows inside one transaction — shared by the
// Dash export import and the external-config importers.
func (s *Server) replaceBoard(sections []Section, settings map[string]json.RawMessage) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, q := range []string{`DELETE FROM urls`, `DELETE FROM items`, `DELETE FROM sections`} {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}
	for _, sec := range sections {
		if sec.ID == "" || sec.Name == "" {
			return fmt.Errorf("%w: section missing id or name", errBadInput)
		}
		if _, err := tx.Exec(`INSERT INTO sections (id, name, position, collapsed) VALUES (?, ?, ?, ?)`,
			sec.ID, sec.Name, sec.Position, sec.Collapsed); err != nil {
			return err
		}
		for _, it := range sec.Items {
			if it.ID == "" || it.Name == "" {
				return fmt.Errorf("%w: item missing id or name", errBadInput)
			}
			var cfg any
			if it.Config != nil {
				cfg = string(*it.Config)
			}
			if _, err := tx.Exec(
				`INSERT INTO items (id, section_id, kind, name, icon, position, config) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				it.ID, sec.ID, orDefault(it.Kind, "service"), it.Name, it.Icon, it.Position, cfg); err != nil {
				return err
			}
			for _, u := range it.URLs {
				if !validURL(u.URL) {
					return fmt.Errorf("%w: invalid url: %s", errBadInput, u.URL)
				}
				uid := u.ID
				if uid == "" {
					uid = newID("u")
				}
				if _, err := tx.Exec(`INSERT INTO urls (id, item_id, url, label, position) VALUES (?, ?, ?, ?, ?)`,
					uid, it.ID, u.URL, u.Label, u.Position); err != nil {
					return err
				}
			}
		}
	}
	for k, v := range settings {
		if _, err := tx.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, string(v)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
