package widget

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Autobrr uses X-Api-Token auth (not X-Api-Key) and a different API layout:
// /api/config for version, /api/release/stats for counters, /api/irc for
// network health, /api/updates/latest answering 200/204. Follows dashbrr's
// internal/services/autobrr.
type autobrrWidget struct{}

func init() { Register(autobrrWidget{}) }

func (autobrrWidget) Meta() Type {
	return Type{
		Type:        "autobrr",
		Name:        "Autobrr",
		Description: "Release stats, IRC health and version",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://autobrr.local:7474", Required: true},
			{Key: "apiKey", Label: "API key", Secret: true, Required: true},
		},
	}
}

func (autobrrWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	key := cfgStr(cfg, "apiKey")
	if endpoint == "" || key == "" {
		return nil, fmt.Errorf("endpoint and apiKey required")
	}
	hdr := map[string]string{"X-Api-Token": key}
	get := func(path string, out any) error {
		return getJSON(ctx, endpoint+path, hdr, out)
	}

	// Config doubles as the liveness + auth probe.
	var ver struct {
		Version string `json:"version"`
	}
	if err := get("/api/config", &ver); err != nil {
		return nil, err
	}

	d := &svcData{Status: "online", Version: ver.Version}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		var stats struct {
			TotalCount        int `json:"total_count"`
			PushApprovedCount int `json:"push_approved_count"`
			PushErrorCount    int `json:"push_error_count"`
		}
		if get("/api/release/stats", &stats) == nil {
			d.Releases = stats.TotalCount
			if stats.PushErrorCount > 0 {
				d.Issues++
				d.Warning = fmt.Sprintf("%d push errors", stats.PushErrorCount)
			}
		}
	}()
	go func() {
		defer wg.Done()
		// /api/irc answers either an array or a single object.
		var irc []struct {
			Name    string `json:"name"`
			Healthy bool   `json:"healthy"`
			Enabled bool   `json:"enabled"`
		}
		var single struct {
			Name    string `json:"name"`
			Healthy bool   `json:"healthy"`
			Enabled bool   `json:"enabled"`
		}
		if get("/api/irc", &irc) == nil {
			for _, s := range irc {
				if s.Enabled && !s.Healthy {
					d.Issues++
					if d.Warning == "" {
						d.Warning = "IRC " + s.Name + " unhealthy"
					}
				}
			}
			return
		}
		if get("/api/irc", &single) == nil && single.Enabled && !single.Healthy {
			d.Issues++
			d.Warning = "IRC " + single.Name + " unhealthy"
		}
	}()
	go func() {
		defer wg.Done()
		// 200 = update available, 204 = none; anything else is ignored.
		code, _, err := getRaw(ctx, endpoint+"/api/updates/latest", hdr)
		if err == nil && code == 200 {
			d.UpdateAvailable = true
		}
	}()
	wg.Wait()

	if d.Issues > 0 {
		d.Status = "warning"
	}
	return d, nil
}
