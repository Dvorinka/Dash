package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// Allowed icon extensions — whitelist keeps the upload dir free of HTML/JS.
var iconExts = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {},
	".gif": {}, ".svg": {}, ".ico": {}, ".avif": {},
}

// uploadIcon stores a multipart file under iconsDir as <itemID><ext> and
// points the item's icon field at /api/icons/<file>.
func (s *Server) uploadIcon(c *gin.Context) {
	id := c.Param("id")
	var exists int
	if err := s.db.QueryRow(`SELECT 1 FROM items WHERE id = ?`, id).Scan(&exists); err != nil {
		fail(c, http.StatusNotFound, "item not found")
		return
	}

	fh, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "file field required")
		return
	}
	if fh.Size > 4<<20 {
		fail(c, http.StatusBadRequest, "icon too large (max 4 MB)")
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if _, ok := iconExts[ext]; !ok {
		fail(c, http.StatusBadRequest, "unsupported icon type")
		return
	}

	name := id + ext
	if err := c.SaveUploadedFile(fh, filepath.Join(s.iconsDir, name)); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.db.Exec(`UPDATE items SET icon = ? WHERE id = ?`, "/api/icons/"+name, id); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	it, err := s.loadItem(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, it)
}

// getIcon serves a stored icon. filepath.Base strips any traversal before
// the path is pinned inside iconsDir.
func (s *Server) getIcon(c *gin.Context) {
	name := filepath.Base(c.Param("file"))
	http.ServeFile(c.Writer, c.Request, filepath.Join(s.iconsDir, name))
}
