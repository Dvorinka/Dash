package widget

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Bazarr uses X-API-KEY auth and wraps payloads in a {data: ...} envelope:
// /api/system/status for the version, /api/system/health for issues.
// Follows dashbrr's internal/services/bazarr.
type bazarrWidget struct{}

func init() { Register(bazarrWidget{}) }

func (bazarrWidget) Meta() Type {
	return Type{
		Type:        "bazarr",
		Name:        "Bazarr",
		Description: "Subtitle health and version",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://bazarr.local:6767", Required: true},
			{Key: "apiKey", Label: "API key", Secret: true, Required: true},
		},
	}
}

func (bazarrWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	key := cfgStr(cfg, "apiKey")
	if endpoint == "" || key == "" {
		return nil, fmt.Errorf("endpoint and apiKey required")
	}
	hdr := map[string]string{"X-API-KEY": key}
	get := func(path string, out any) error {
		return getJSON(ctx, endpoint+path, hdr, out)
	}

	var st struct {
		Data struct {
			Version string `json:"bazarr_version"`
		} `json:"data"`
	}
	if err := get("/api/system/status", &st); err != nil {
		return nil, err
	}

	d := &svcData{Status: "online", Version: st.Data.Version}

	var health struct {
		Data []struct {
			Object string `json:"object"`
			Issue  string `json:"issue"`
		} `json:"data"`
	}
	if get("/api/system/health", &health) == nil {
		for _, h := range health.Data {
			d.Issues++
			if d.Warning == "" {
				d.Warning = "[" + h.Object + "] " + h.Issue
			}
		}
	}
	if d.Issues > 0 {
		d.Status = "warning"
	}
	return d, nil
}
