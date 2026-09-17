package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// statusEntry is one cached reachability probe result.
type statusEntry struct {
	Status    string    `json:"status"` // "up" | "down"
	LatencyMs int64     `json:"latencyMs"`
	CheckedAt time.Time `json:"checkedAt"`
}

// statusCache is a TTL map of url -> probe result. Pings are HEAD requests;
// any HTTP response < 500 means the host answered, so the service is up.
// ponytail: pings the primary URL only — extend to all urls if demanded.
type statusCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]statusEntry
	client  *http.Client
}

func newStatusCache(ttl time.Duration) *statusCache {
	return &statusCache{
		ttl:     ttl,
		entries: map[string]statusEntry{},
		client:  &http.Client{Timeout: 4 * time.Second},
	}
}

// check returns the cached probe if fresh, otherwise pings synchronously.
func (sc *statusCache) check(url string) statusEntry {
	sc.mu.Lock()
	if e, ok := sc.entries[url]; ok && time.Since(e.CheckedAt) < sc.ttl {
		sc.mu.Unlock()
		return e
	}
	sc.mu.Unlock()

	e := sc.ping(url)

	sc.mu.Lock()
	sc.entries[url] = e
	sc.mu.Unlock()
	return e
}

func (sc *statusCache) ping(url string) statusEntry {
	start := time.Now()
	e := statusEntry{Status: "down", CheckedAt: start}

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return e
	}
	resp, err := sc.client.Do(req)
	e.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 500 {
		e.Status = "up"
	}
	return e
}

// getStatus probes the primary URL of every service item, in parallel,
// and returns a map keyed by item id.
func (s *Server) getStatus(c *gin.Context) {
	rows, err := s.db.Query(
		`SELECT i.id, u.url FROM items i
		 JOIN urls u ON u.item_id = i.id
		 WHERE i.kind = 'service'
		   AND u.position = (SELECT MIN(position) FROM urls WHERE item_id = i.id)`)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	type target struct{ id, url string }
	var targets []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.id, &t.url); err != nil {
			rows.Close()
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		targets = append(targets, t)
	}
	rows.Close()

	out := make(map[string]statusEntry, len(targets))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, t := range targets {
		wg.Add(1)
		go func(t target) {
			defer wg.Done()
			e := s.status.check(t.url)
			mu.Lock()
			out[t.id] = e
			mu.Unlock()
		}(t)
	}
	wg.Wait()
	c.JSON(http.StatusOK, out)
}
