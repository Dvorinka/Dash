package widget

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Media servers — Jellyfin, Emby, Plex — share the shape: a /System/Info-style
// version endpoint plus a sessions endpoint for active streams. Auth and path
// prefixes differ per product; patterns follow dashbrr's jellyfin/plex svcs.
type mediaWidget struct {
	typ     string
	name    string
	ph      string
	prefix  string // emby mounts under /emby
	headers func(apiKey string) map[string]string
	verPath string
	sesPath string
	plex    bool // plex wraps payloads in MediaContainer and needs client headers
}

func (w mediaWidget) Meta() Type {
	return Type{
		Type:        w.typ,
		Name:        w.name,
		Description: w.name + " streams and version",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: w.ph, Required: true},
			{Key: "apiKey", Label: "API key", Secret: true, Required: true},
		},
	}
}

func init() {
	Register(mediaWidget{
		typ: "jellyfin", name: "Jellyfin", ph: "http://jellyfin.local:8096",
		verPath: "/System/Info", sesPath: "/Sessions?ActiveWithinSeconds=300",
		headers: func(k string) map[string]string {
			return map[string]string{"Authorization": fmt.Sprintf("MediaBrowser Token=%q", k)}
		},
	})
	Register(mediaWidget{
		typ: "emby", name: "Emby", ph: "http://emby.local:8096",
		prefix: "/emby", verPath: "/System/Info", sesPath: "/Sessions",
		headers: func(k string) map[string]string { return map[string]string{"X-Emby-Token": k} },
	})
	Register(mediaWidget{
		typ: "plex", name: "Plex", ph: "http://plex.local:32400", plex: true,
		verPath: "/identity", sesPath: "/status/sessions",
		headers: func(k string) map[string]string {
			return map[string]string{
				"X-Plex-Token":             k,
				"X-Plex-Client-Identifier": "com.dash.client",
				"X-Plex-Product":           "Dash",
				"X-Plex-Version":           "1.0.0",
				"X-Plex-Platform":          "Web",
				"X-Plex-Device":            "Browser",
			}
		},
	})
}

func (w mediaWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	key := cfgStr(cfg, "apiKey")
	if endpoint == "" || key == "" {
		return nil, fmt.Errorf("endpoint and apiKey required")
	}
	if w.prefix != "" {
		endpoint = strings.TrimSuffix(endpoint, w.prefix) + w.prefix
	}
	hdr := w.headers(key)

	d := &svcData{Status: "online"}

	if w.plex {
		var id struct {
			MediaContainer struct {
				Version      string `json:"version"`
				FriendlyName string `json:"friendlyName"`
			} `json:"MediaContainer"`
		}
		if err := getJSON(ctx, endpoint+w.verPath, hdr, &id); err != nil {
			return nil, err
		}
		d.Version = id.MediaContainer.Version
		d.Name = id.MediaContainer.FriendlyName
	} else {
		var info struct {
			Version     string `json:"Version"`
			ServerName  string `json:"ServerName"`
			ProductName string `json:"ProductName"`
		}
		if err := getJSON(ctx, endpoint+w.verPath, hdr, &info); err != nil {
			return nil, err
		}
		d.Version = info.Version
		d.Name = info.ServerName
		if d.Name == "" {
			d.Name = info.ProductName
		}
	}

	// Sessions are best-effort — a server without session access still reports.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if w.plex {
			var s struct {
				MediaContainer struct {
					Size int `json:"size"`
				} `json:"MediaContainer"`
			}
			if getJSON(ctx, endpoint+w.sesPath, hdr, &s) == nil {
				d.Streams = s.MediaContainer.Size
			}
			return
		}
		var sessions []json.RawMessage
		if getJSON(ctx, endpoint+w.sesPath, hdr, &sessions) == nil {
			d.Streams = len(sessions)
		}
	}()
	wg.Wait()
	return d, nil
}
