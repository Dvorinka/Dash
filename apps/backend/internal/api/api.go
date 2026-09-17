// Package api wires the Gin router and /api handlers.
package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter builds the HTTP handler: zap access log, recovery, /api routes.
func NewRouter(logger *zap.Logger, db *sql.DB) *gin.Engine {
	r := gin.New()
	r.Use(accessLog(logger), gin.Recovery())

	v1 := r.Group("/api")
	v1.GET("/healthz", func(c *gin.Context) {
		if err := db.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unreachable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}

// accessLog is gin.Logger backed by zap: one structured line per request.
func accessLog(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}
