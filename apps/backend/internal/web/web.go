// Package web serves the embedded frontend build (SPA with index.html
// fallback). dist/ is populated by the frontend build in Docker/CI; the
// committed .keep keeps //go:embed satisfied on a bare clone.
package web

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var dist embed.FS

// Register mounts the SPA on the router. Non-/api paths serve static files;
// unknown paths fall back to index.html for client-side routing.
func Register(r *gin.Engine) error {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return err
	}
	fileServer := http.FileServer(http.FS(sub))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if f, err := sub.Open(path[1:]); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
	return nil
}
