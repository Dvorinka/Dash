package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type itemInput struct {
	SectionID string           `json:"sectionId"`
	Kind      string           `json:"kind"`
	Name      string           `json:"name"`
	Icon      string           `json:"icon"`
	Config    *json.RawMessage `json:"config"`
	URLs      []URLInput       `json:"urls"`
}

func (s *Server) createItem(c *gin.Context) {
	var in itemInput
	if err := c.ShouldBindJSON(&in); err != nil || in.SectionID == "" || in.Name == "" {
		fail(c, http.StatusBadRequest, "sectionId and name required")
		return
	}
	kind := in.Kind
	if kind == "" {
		kind = "service"
	}
	if kind != "service" && kind != "widget" {
		fail(c, http.StatusBadRequest, "kind must be service or widget")
		return
	}
	for _, u := range in.URLs {
		if !validURL(u.URL) {
			fail(c, http.StatusBadRequest, "invalid url: "+u.URL)
			return
		}
	}

	var maxPos sql.NullFloat64
	err := s.db.QueryRow(`SELECT MAX(position) FROM items WHERE section_id = ?`, in.SectionID).Scan(&maxPos)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	it := Item{ID: newID("i"), SectionID: in.SectionID, Kind: kind, Name: in.Name, Icon: in.Icon,
		Config: in.Config, Position: midpoint(ptrOr(maxPos), nil), URLs: []URL{}}

	tx, err := s.db.Begin()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback() }()

	var cfg any
	if in.Config != nil {
		cfg = string(*in.Config)
	}
	if _, err := tx.Exec(`INSERT INTO items (id, section_id, kind, name, icon, position, config) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		it.ID, it.SectionID, it.Kind, it.Name, it.Icon, it.Position, cfg); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := insertURLs(tx, it.ID, in.URLs, &it.URLs); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, it)
}

// insertURLs writes the url list (position = index * 1024) and fills out.
func insertURLs(tx *sql.Tx, itemID string, in []URLInput, out *[]URL) error {
	for i, u := range in {
		row := URL{ID: newID("u"), URL: u.URL, Label: u.Label, Position: float64(i+1) * 1024}
		if _, err := tx.Exec(`INSERT INTO urls (id, item_id, url, label, position) VALUES (?, ?, ?, ?, ?)`,
			row.ID, itemID, row.URL, row.Label, row.Position); err != nil {
			return err
		}
		*out = append(*out, row)
	}
	return nil
}

func (s *Server) updateItem(c *gin.Context) {
	var in struct {
		Name   *string          `json:"name"`
		Icon   *string          `json:"icon"`
		Config *json.RawMessage `json:"config"`
		URLs   *[]URLInput      `json:"urls"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	if in.URLs != nil {
		for _, u := range *in.URLs {
			if !validURL(u.URL) {
				fail(c, http.StatusBadRequest, "invalid url: "+u.URL)
				return
			}
		}
	}
	id := c.Param("id")

	tx, err := s.db.Begin()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback() }()

	if in.Name != nil {
		if _, err := tx.Exec(`UPDATE items SET name = ? WHERE id = ?`, *in.Name, id); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if in.Icon != nil {
		if _, err := tx.Exec(`UPDATE items SET icon = ? WHERE id = ?`, *in.Icon, id); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if in.Config != nil {
		if _, err := tx.Exec(`UPDATE items SET config = ? WHERE id = ?`, string(*in.Config), id); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if in.URLs != nil {
		if _, err := tx.Exec(`DELETE FROM urls WHERE item_id = ?`, id); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		var out []URL
		if err := insertURLs(tx, id, *in.URLs, &out); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if err := tx.Commit(); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	it, err := s.loadItem(id)
	if notFound(err) {
		fail(c, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, it)
}

// loadItem reads one item with its urls.
func (s *Server) loadItem(id string) (Item, error) {
	var it Item
	var cfg sql.NullString
	err := s.db.QueryRow(`SELECT id, section_id, kind, name, icon, position, config FROM items WHERE id = ?`, id).
		Scan(&it.ID, &it.SectionID, &it.Kind, &it.Name, &it.Icon, &it.Position, &cfg)
	if err != nil {
		return it, err
	}
	if cfg.Valid {
		raw := json.RawMessage(cfg.String)
		it.Config = &raw
	}
	it.URLs = []URL{}
	rows, err := s.db.Query(`SELECT id, url, label, position FROM urls WHERE item_id = ? ORDER BY position`, id)
	if err != nil {
		return it, err
	}
	defer rows.Close()
	for rows.Next() {
		var u URL
		if err := rows.Scan(&u.ID, &u.URL, &u.Label, &u.Position); err != nil {
			return it, err
		}
		it.URLs = append(it.URLs, u)
	}
	return it, rows.Err()
}

func (s *Server) deleteItem(c *gin.Context) {
	res, err := s.db.Exec(`DELETE FROM items WHERE id = ?`, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "item not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) reorderItem(c *gin.Context) {
	var in struct {
		ID        string  `json:"id"`
		SectionID *string `json:"sectionId"`
		BeforeID  *string `json:"beforeId"`
		AfterID   *string `json:"afterId"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ID == "" {
		fail(c, http.StatusBadRequest, "id required")
		return
	}
	pos, err := s.neighborPosition("items", in.BeforeID, in.AfterID)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if in.SectionID == nil {
		if err := s.db.QueryRow(`SELECT section_id FROM items WHERE id = ?`, in.ID).Scan(&in.SectionID); notFound(err) {
			fail(c, http.StatusNotFound, "item not found")
			return
		} else if err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	res, err := s.db.Exec(`UPDATE items SET position = ?, section_id = ? WHERE id = ?`, pos, *in.SectionID, in.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "item not found")
		return
	}
	it, err := s.loadItem(in.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, it)
}
