package api

// Multiple boards — sections carry board_id; the board list is small and
// lives next to the section tree. Deleting a board cascades its sections.

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Board is a named dashboard page reachable at /b/<slug>.
type Board struct {
	ID       string  `json:"id"`
	Slug     string  `json:"slug"`
	Name     string  `json:"name"`
	Position float64 `json:"position"`
}

func (s *Server) listBoards(c *gin.Context) {
	rows, err := s.db.Query(`SELECT id, slug, name, position FROM boards ORDER BY position`)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []Board{}
	for rows.Next() {
		var b Board
		if err := rows.Scan(&b.ID, &b.Slug, &b.Name, &b.Position); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, b)
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) createBoard(c *gin.Context) {
	var in struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		fail(c, http.StatusBadRequest, "name required")
		return
	}
	name := strings.TrimSpace(in.Name)
	slug := slugify(name)
	if slug == "" {
		slug = "board"
	}
	// Uniquify: 'media', 'media-2', 'media-3'…
	base := slug
	for i := 2; ; i++ {
		var n int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM boards WHERE slug = ?`, slug).Scan(&n)
		if n == 0 {
			break
		}
		slug = base + "-" + strconv.Itoa(i)
	}
	var maxPos sql.NullFloat64
	_ = s.db.QueryRow(`SELECT MAX(position) FROM boards`).Scan(&maxPos)
	b := Board{ID: newID("b"), Slug: slug, Name: name, Position: midpoint(ptrOr(maxPos), nil)}
	if _, err := s.db.Exec(`INSERT INTO boards (id, slug, name, position) VALUES (?,?,?,?)`,
		b.ID, b.Slug, b.Name, b.Position); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, b)
}

func (s *Server) updateBoard(c *gin.Context) {
	var in struct {
		Name *string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Name == nil || strings.TrimSpace(*in.Name) == "" {
		fail(c, http.StatusBadRequest, "name required")
		return
	}
	// Slug stays fixed on rename — links shouldn't break.
	res, err := s.db.Exec(`UPDATE boards SET name = ? WHERE id = ?`, strings.TrimSpace(*in.Name), c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "board not found")
		return
	}
	var b Board
	if err := s.db.QueryRow(`SELECT id, slug, name, position FROM boards WHERE id = ?`, c.Param("id")).
		Scan(&b.ID, &b.Slug, &b.Name, &b.Position); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, b)
}

func (s *Server) deleteBoard(c *gin.Context) {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM boards`).Scan(&n)
	if n <= 1 {
		fail(c, http.StatusBadRequest, "cannot delete the last board")
		return
	}
	res, err := s.db.Exec(`DELETE FROM boards WHERE id = ?`, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "board not found")
		return
	}
	c.Status(http.StatusNoContent)
}
