package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdvorak/dash/internal/widget"
)

// widgetEntry is one cached fetch result.
type widgetEntry struct {
	Data      any       `json:"data"`
	FetchedAt time.Time `json:"fetchedAt"`
	err       error
}

// widgetCache is a TTL map of item id -> fetch result. Local widgets never
// reach this — the frontend renders them without a fetch.
type widgetCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]widgetEntry
}

func newWidgetCache(ttl time.Duration) *widgetCache {
	return &widgetCache{ttl: ttl, entries: map[string]widgetEntry{}}
}

func (s *Server) widgetTypes(c *gin.Context) {
	c.JSON(http.StatusOK, widget.Types())
}

// widgetData resolves the item's config.type, fetches via the registry,
// and caches per item for ~30s. Fetch errors become 502 with only the error
// message — config values (API keys) must never leak into a response.
func (s *Server) widgetData(c *gin.Context) {
	id := c.Param("id")

	s.widgets.mu.Lock()
	if e, ok := s.widgets.entries[id]; ok && time.Since(e.FetchedAt) < s.widgets.ttl {
		s.widgets.mu.Unlock()
		if e.err != nil {
			c.JSON(http.StatusBadGateway, e.Data)
			return
		}
		c.JSON(http.StatusOK, e)
		return
	}
	s.widgets.mu.Unlock()

	it, err := s.loadItem(id)
	if notFound(err) {
		fail(c, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if it.Kind != "widget" || it.Config == nil {
		fail(c, http.StatusBadRequest, "not a widget item")
		return
	}
	var cfg struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(*it.Config, &cfg); err != nil || cfg.Type == "" {
		fail(c, http.StatusBadRequest, "widget config missing type")
		return
	}
	f, ok := widget.Get(cfg.Type)
	if !ok {
		fail(c, http.StatusBadRequest, "unknown widget type: "+cfg.Type)
		return
	}

	data, err := f.Fetch(c.Request.Context(), *it.Config)
	e := widgetEntry{FetchedAt: time.Now()}
	if err != nil {
		// Cache failures briefly too — a dead upstream should not turn
		// every poll into a live request.
		e.Data = gin.H{"error": err.Error()}
		e.err = err
	} else {
		e.Data = data
	}

	s.widgets.mu.Lock()
	s.widgets.entries[id] = e
	s.widgets.mu.Unlock()

	if e.err != nil {
		c.JSON(http.StatusBadGateway, e.Data)
		return
	}
	c.JSON(http.StatusOK, e)
}
