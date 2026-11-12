package widget

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// svcData is the flat payload every service widget returns. The frontend
// ServiceWidget renders whichever fields are set.
type svcData struct {
	Status          string `json:"status"` // online | warning
	Version         string `json:"version,omitempty"`
	Name            string `json:"name,omitempty"`
	QueueTotal      int    `json:"queueTotal,omitempty"`
	PendingRequests int    `json:"pendingRequests,omitempty"`
	Indexers        int    `json:"indexers,omitempty"`
	Releases        int    `json:"releases,omitempty"`
	Streams         int    `json:"streams,omitempty"`
	Up              int    `json:"up"` // zero is meaningful: 0 of N running
	Total           int    `json:"total,omitempty"`
	Speed           string `json:"speed,omitempty"`
	UpdateAvailable bool   `json:"updateAvailable,omitempty"`
	Paused          bool   `json:"paused,omitempty"`
	Issues          int    `json:"issues"`
	Warning         string `json:"warning,omitempty"`
}

// arrWidget covers the *arr family — Sonarr, Radarr, Lidarr, Readarr,
// Whisparr, Prowlarr — which share one API shape: X-Api-Key auth and
// /api/<ver>/<resource> paths (v3 for the movie/TV apps, v1 for the rest).
// Pattern follows dashbrr's internal/services/arr.
type arrWidget struct {
	typ     string
	name    string
	ph      string // endpoint placeholder with default port
	apiV    string // path segment: "v3" or "v1"
	queue   bool   // exposes /api/<v>/queue
	indexer bool   // prowlarr: count /api/v1/indexer instead of a queue
}

func (w arrWidget) Meta() Type {
	return Type{
		Type:        w.typ,
		Name:        w.name,
		Description: w.name + " health, version and queue",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: w.ph, Required: true},
			{Key: "apiKey", Label: "API key", Secret: true, Required: true},
		},
	}
}

func init() {
	for _, w := range []arrWidget{
		{typ: "sonarr", name: "Sonarr", ph: "http://sonarr.local:8989", apiV: "v3", queue: true},
		{typ: "radarr", name: "Radarr", ph: "http://radarr.local:7878", apiV: "v3", queue: true},
		{typ: "lidarr", name: "Lidarr", ph: "http://lidarr.local:8686", apiV: "v1", queue: true},
		{typ: "readarr", name: "Readarr", ph: "http://readarr.local:8787", apiV: "v1", queue: true},
		{typ: "whisparr", name: "Whisparr", ph: "http://whisparr.local:6969", apiV: "v3", queue: true},
		{typ: "prowlarr", name: "Prowlarr", ph: "http://prowlarr.local:9696", apiV: "v1", indexer: true},
	} {
		Register(w)
	}
	Register(reqWidget{typ: "overseerr", name: "Overseerr", ph: "http://overseerr.local:5055"})
	Register(reqWidget{typ: "jellyseerr", name: "Jellyseerr", ph: "http://jellyseerr.local:5055"})
}

// arrIssue is one entry of the *arr /health response.
type arrIssue struct {
	Type    string `json:"type"`
	Source  string `json:"source"`
	Message string `json:"message"`
}

func (w arrWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	key := cfgStr(cfg, "apiKey")
	if endpoint == "" || key == "" {
		return nil, fmt.Errorf("endpoint and apiKey required")
	}
	hdr := map[string]string{"X-Api-Key": key}
	get := func(path string, out any) error {
		return getJSON(ctx, endpoint+"/api/"+w.apiV+path, hdr, out)
	}

	// Health doubles as the liveness probe — its failure fails the widget.
	var issues []arrIssue
	if err := get("/health", &issues); err != nil {
		return nil, err
	}

	d := &svcData{Status: "online"}
	for _, is := range issues {
		if is.Type != "warning" && is.Type != "error" {
			continue
		}
		d.Issues++
		if d.Warning == "" {
			d.Warning = "[" + is.Source + "] " + is.Message
		}
	}
	if d.Issues > 0 {
		d.Status = "warning"
	}

	// Enrichment is best-effort and concurrent — a missing endpoint on an
	// older version must not sink the whole widget.
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		var st struct {
			Version string `json:"version"`
		}
		if get("/system/status", &st) == nil {
			d.Version = st.Version
		}
	}()
	go func() {
		defer wg.Done()
		switch {
		case w.indexer:
			var idx []struct {
				ID int `json:"id"`
			}
			if get("/indexer", &idx) == nil {
				d.Indexers = len(idx)
			}
		case w.queue:
			var q struct {
				TotalRecords int `json:"totalRecords"`
			}
			if get("/queue?page=1&pageSize=1", &q) == nil {
				d.QueueTotal = q.TotalRecords
			}
		}
	}()
	go func() {
		defer wg.Done()
		var ups []struct {
			Installed   bool `json:"installed"`
			Installable bool `json:"installable"`
		}
		if get("/update", &ups) == nil {
			for _, u := range ups {
				if u.Installable && !u.Installed {
					d.UpdateAvailable = true
				}
			}
		}
	}()
	wg.Wait()
	return d, nil
}

// reqWidget covers Overseerr and Jellyseerr, which share the same API:
// X-Api-Key auth, /api/v1/status for version, /api/v1/request for the
// pending-request count (same take=10 window dashbrr uses).
type reqWidget struct {
	typ  string
	name string
	ph   string
}

func (w reqWidget) Meta() Type {
	return Type{
		Type:        w.typ,
		Name:        w.name,
		Description: w.name + " requests and version",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: w.ph, Required: true},
			{Key: "apiKey", Label: "API key", Secret: true, Required: true},
		},
	}
}

func (w reqWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	key := cfgStr(cfg, "apiKey")
	if endpoint == "" || key == "" {
		return nil, fmt.Errorf("endpoint and apiKey required")
	}
	hdr := map[string]string{"X-Api-Key": key}

	var st struct {
		Version         string `json:"version"`
		UpdateAvailable bool   `json:"updateAvailable"`
	}
	if err := getJSON(ctx, endpoint+"/api/v1/status", hdr, &st); err != nil {
		return nil, err
	}

	var reqs struct {
		Results []struct {
			Status int `json:"status"` // 1 = pending
		} `json:"results"`
	}
	pending := 0
	if getJSON(ctx, endpoint+"/api/v1/request?take=10", hdr, &reqs) == nil {
		for _, r := range reqs.Results {
			if r.Status == 1 {
				pending++
			}
		}
	}

	return &svcData{
		Status:          "online",
		Version:         st.Version,
		PendingRequests: pending,
		UpdateAvailable: st.UpdateAvailable,
		Issues:          0,
	}, nil
}
