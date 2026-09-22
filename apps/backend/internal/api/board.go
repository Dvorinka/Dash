package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// board loads the full tree: sections ordered, each with items and urls.
// Three flat queries assembled in memory — the whole board is ~100 rows.
// boardID filters to one board; "" returns every section (export, legacy).
// COALESCE maps pre-boards rows onto the default 'b_home' board.
func (s *Server) board(boardID string) ([]Section, error) {
	q := `SELECT id, name, position, collapsed, COALESCE(board_id,'b_home') FROM sections`
	var args []any
	if boardID != "" {
		q += ` WHERE board_id = ?`
		args = append(args, boardID)
	}
	secRows, err := s.db.Query(q+` ORDER BY position`, args...)
	if err != nil {
		return nil, err
	}
	defer secRows.Close()

	sections := []Section{}
	byID := map[string]*Section{}
	for secRows.Next() {
		var sec Section
		if err := secRows.Scan(&sec.ID, &sec.Name, &sec.Position, &sec.Collapsed, &sec.BoardID); err != nil {
			return nil, err
		}
		sec.Items = []Item{}
		sections = append(sections, sec)
	}
	for i := range sections {
		byID[sections[i].ID] = &sections[i]
	}
	if err := secRows.Err(); err != nil {
		return nil, err
	}

	itemRows, err := s.db.Query(`SELECT id, section_id, kind, name, icon, position, config FROM items ORDER BY position`)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	items := []*Item{}
	itemByID := map[string]*Item{}
	for itemRows.Next() {
		var it Item
		var cfg sql.NullString
		if err := itemRows.Scan(&it.ID, &it.SectionID, &it.Kind, &it.Name, &it.Icon, &it.Position, &cfg); err != nil {
			return nil, err
		}
		if cfg.Valid {
			raw := json.RawMessage(cfg.String)
			it.Config = &raw
		}
		it.URLs = []URL{}
		items = append(items, &it)
		itemByID[it.ID] = &it
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	urlRows, err := s.db.Query(`SELECT id, item_id, url, label, position FROM urls ORDER BY position`)
	if err != nil {
		return nil, err
	}
	defer urlRows.Close()

	for urlRows.Next() {
		var u URL
		var itemID string
		if err := urlRows.Scan(&u.ID, &itemID, &u.URL, &u.Label, &u.Position); err != nil {
			return nil, err
		}
		if it, ok := itemByID[itemID]; ok {
			it.URLs = append(it.URLs, u)
		}
	}
	if err := urlRows.Err(); err != nil {
		return nil, err
	}

	for _, it := range items {
		if sec, ok := byID[it.SectionID]; ok {
			sec.Items = append(sec.Items, *it)
		}
	}
	return sections, nil
}

func (s *Server) listSections(c *gin.Context) {
	sections, err := s.board(c.Query("board"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, sections)
}

func (s *Server) createSection(c *gin.Context) {
	var in struct {
		Name    string `json:"name"`
		BoardID string `json:"boardId"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Name == "" {
		fail(c, http.StatusBadRequest, "name required")
		return
	}
	boardID := in.BoardID
	if boardID == "" {
		boardID = "b_home"
	}
	var maxPos sql.NullFloat64
	if err := s.db.QueryRow(`SELECT MAX(position) FROM sections WHERE board_id = ?`, boardID).Scan(&maxPos); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	sec := Section{ID: newID("s"), Name: in.Name, Position: midpoint(ptrOr(maxPos), nil), BoardID: boardID, Items: []Item{}}
	// The subquery yields NULL for an unknown board id instead of an FK error;
	// board() maps NULL back onto the default board.
	if _, err := s.db.Exec(`INSERT INTO sections (id, name, position, board_id)
		VALUES (?, ?, ?, (SELECT id FROM boards WHERE id = ?))`, sec.ID, sec.Name, sec.Position, boardID); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, sec)
}

func (s *Server) updateSection(c *gin.Context) {
	var in struct {
		Name      *string `json:"name"`
		Collapsed *bool   `json:"collapsed"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Name != nil {
		if _, err := s.db.Exec(`UPDATE sections SET name = ? WHERE id = ?`, *in.Name, c.Param("id")); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if in.Collapsed != nil {
		if _, err := s.db.Exec(`UPDATE sections SET collapsed = ? WHERE id = ?`, *in.Collapsed, c.Param("id")); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	var sec Section
	err := s.db.QueryRow(`SELECT id, name, position, collapsed, COALESCE(board_id,'b_home') FROM sections WHERE id = ?`, c.Param("id")).
		Scan(&sec.ID, &sec.Name, &sec.Position, &sec.Collapsed, &sec.BoardID)
	if notFound(err) {
		fail(c, http.StatusNotFound, "section not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	sec.Items = []Item{}
	c.JSON(http.StatusOK, sec)
}

func (s *Server) deleteSection(c *gin.Context) {
	res, err := s.db.Exec(`DELETE FROM sections WHERE id = ?`, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "section not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) reorderSection(c *gin.Context) {
	var in struct {
		ID       string  `json:"id"`
		BeforeID *string `json:"beforeId"`
		AfterID  *string `json:"afterId"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ID == "" {
		fail(c, http.StatusBadRequest, "id required")
		return
	}
	pos, err := s.neighborPosition("sections", in.BeforeID, in.AfterID)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	res, err := s.db.Exec(`UPDATE sections SET position = ? WHERE id = ?`, pos, in.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "section not found")
		return
	}
	var sec Section
	if err := s.db.QueryRow(`SELECT id, name, position, collapsed, COALESCE(board_id,'b_home') FROM sections WHERE id = ?`, in.ID).
		Scan(&sec.ID, &sec.Name, &sec.Position, &sec.Collapsed, &sec.BoardID); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	sec.Items = []Item{}
	c.JSON(http.StatusOK, sec)
}

// neighborPosition resolves the new fractional position between two neighbors.
func (s *Server) neighborPosition(table string, beforeID, afterID *string) (float64, error) {
	var before, after *float64
	if beforeID != nil {
		p, err := s.positionOf(table, *beforeID)
		if err != nil {
			return 0, err
		}
		before = &p
	}
	if afterID != nil {
		p, err := s.positionOf(table, *afterID)
		if err != nil {
			return 0, err
		}
		after = &p
	}
	return midpoint(before, after), nil
}

func (s *Server) positionOf(table, id string) (float64, error) {
	var p float64
	// table is an internal constant (sections|items) — never user input.
	err := s.db.QueryRow(`SELECT position FROM `+table+` WHERE id = ?`, id).Scan(&p)
	return p, err
}

func ptrOr(v sql.NullFloat64) *float64 {
	if v.Valid {
		return &v.Float64
	}
	return nil
}
