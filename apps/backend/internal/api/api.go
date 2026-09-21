// Package api wires the Gin router and /api handlers.
package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdvorak/dash/internal/domain"
	"github.com/tdvorak/dash/internal/widget"
	"go.uber.org/zap"
)

// Server holds handler dependencies: DB, icon dir, logger, caches.
type Server struct {
	db       *sql.DB
	iconsDir string
	log      *zap.Logger
	status   *statusCache
	widgets  *widgetCache
	// lookup performs domain refreshes; swappable so tests don't hit the
	// real WHOIS/RDAP/DNS path.
	lookup func(ctx context.Context, name string) *domain.Result
}

func newServer(logger *zap.Logger, db *sql.DB, iconsDir string) *Server {
	s := &Server{
		db:       db,
		iconsDir: iconsDir,
		log:      logger,
		status:   newStatusCache(60 * time.Second),
		widgets:  newWidgetCache(30 * time.Second),
	}
	s.lookup = domain.NewLookup().Run
	widget.Register(&monitorWidget{db: db})
	widget.Register(&domainWidget{db: db})
	widget.Register(&systemWidget{db: db})
	return s
}

// NewRouter builds the HTTP handler: zap access log, recovery, /api routes.
// A non-nil ctx starts the monitor scheduler; nil keeps it off (tests).
func NewRouter(ctx context.Context, logger *zap.Logger, db *sql.DB, iconsDir string) *gin.Engine {
	s := newServer(logger, db, iconsDir)
	if ctx != nil {
		go s.runScheduler(ctx.Done())
	}
	return s.routes()
}

func (s *Server) routes() *gin.Engine {
	r := gin.New()
	r.Use(accessLog(s.log), gin.Recovery())

	v1 := r.Group("/api")
	v1.GET("/healthz", s.healthz)

	v1.GET("/sections", s.listSections)
	v1.POST("/sections", s.createSection)
	v1.PATCH("/sections/:id", s.updateSection)
	v1.DELETE("/sections/:id", s.deleteSection)
	v1.POST("/sections/reorder", s.reorderSection)

	v1.POST("/items", s.createItem)
	v1.PATCH("/items/:id", s.updateItem)
	v1.DELETE("/items/:id", s.deleteItem)
	v1.POST("/items/reorder", s.reorderItem)
	v1.POST("/items/:id/icon", s.uploadIcon)
	v1.GET("/icons/:file", s.getIcon)

	v1.GET("/monitors", s.listMonitorsH)
	v1.POST("/monitors", s.createMonitor)
	v1.GET("/monitors/:id", s.getMonitor)
	v1.PATCH("/monitors/:id", s.patchMonitor)
	v1.DELETE("/monitors/:id", s.deleteMonitor)
	v1.GET("/monitors/:id/heartbeats", s.monitorHeartbeats)
	v1.POST("/monitors/:id/check", s.checkNow)
	v1.Any("/push/:token", s.pushIngest)

	v1.GET("/domains", s.listDomainsH)
	v1.POST("/domains", s.createDomain)
	v1.GET("/domains/:id", s.getDomain)
	v1.PATCH("/domains/:id", s.patchDomain)
	v1.DELETE("/domains/:id", s.deleteDomain)
	v1.POST("/domains/:id/refresh", s.refreshDomainH)
	v1.GET("/domains/:id/checks", s.domainChecks)
	v1.POST("/notify/test", s.notifyTest)

	v1.GET("/systems", s.listSystemsH)
	v1.POST("/systems", s.createSystem)
	v1.GET("/systems/:id", s.getSystem)
	v1.PATCH("/systems/:id", s.patchSystem)
	v1.DELETE("/systems/:id", s.deleteSystem)
	v1.GET("/systems/:id/stats", s.systemStats)
	v1.POST("/systems/ingest", s.ingestSystem)

	v1.GET("/incidents", s.listIncidents)
	v1.POST("/incidents", s.createIncident)
	v1.PATCH("/incidents/:id", s.patchIncident)
	v1.DELETE("/incidents/:id", s.deleteIncident)
	v1.POST("/incidents/:id/updates", s.addIncidentUpdate)

	v1.GET("/maintenance", s.listWindows)
	v1.POST("/maintenance", s.createWindow)
	v1.DELETE("/maintenance/:id", s.deleteWindow)

	v1.GET("/status-pages", s.listStatusPages)
	v1.POST("/status-pages", s.createStatusPage)
	v1.PATCH("/status-pages/:id", s.patchStatusPage)
	v1.DELETE("/status-pages/:id", s.deleteStatusPage)
	v1.GET("/status-pages/:slug/public", s.publicStatus)

	v1.GET("/badge/:kind/:file", s.badge)
	v1.GET("/metrics", s.metrics)
	v1.POST("/import/csv", s.importCSV)

	v1.GET("/status", s.getStatus)
	v1.GET("/widgets/types", s.widgetTypes)
	v1.GET("/widgets/:id/data", s.widgetData)
	v1.GET("/settings", s.getSettings)
	v1.PUT("/settings", s.putSettings)
	v1.GET("/export", s.exportBoard)
	v1.POST("/import", s.importBoard)
	v1.POST("/import/external", s.importExternal)

	return r
}

// ---------- shared types (mirror openapi.yaml) ----------

// URL is one launch target of an item (local, external, custom label).
type URL struct {
	ID       string  `json:"id"`
	URL      string  `json:"url"`
	Label    string  `json:"label"`
	Position float64 `json:"position"`
}

// URLInput is a URL as accepted on create/update (id/position assigned).
type URLInput struct {
	URL   string `json:"url"`
	Label string `json:"label"`
}

// Item is a board cell: a service tile today, a widget in Phase 2.
type Item struct {
	ID        string           `json:"id"`
	SectionID string           `json:"sectionId"`
	Kind      string           `json:"kind"`
	Name      string           `json:"name"`
	Icon      string           `json:"icon"`
	Position  float64          `json:"position"`
	Config    *json.RawMessage `json:"config"`
	URLs      []URL            `json:"urls"`
}

// Section is a named, collapsible group of items.
type Section struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Position  float64 `json:"position"`
	Collapsed bool    `json:"collapsed"`
	Items     []Item  `json:"items"`
}

// ---------- helpers ----------

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

// midpoint returns a fractional position between two optional neighbors.
func midpoint(before, after *float64) float64 {
	switch {
	case before == nil && after == nil:
		return 1024
	case before == nil:
		return *after - 1024
	case after == nil:
		return *before + 1024
	default:
		return (*before + *after) / 2
	}
}

func fail(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, gin.H{"error": msg})
}

// validURL enforces http(s) launch targets — also keeps /api/status safe
// from file:// or internal-scheme abuse.
func validURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func notFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }

// healthz reports liveness and DB reachability.
func (s *Server) healthz(c *gin.Context) {
	if err := s.db.PingContext(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unreachable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
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
