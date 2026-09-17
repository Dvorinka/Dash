// Package widget is the integration registry: each widget type declares its
// config fields and a Fetcher that pulls live data from the target service.
// Adding an integration = one file here + one React component.
package widget

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// Field describes one config input rendered by the add-widget dialog.
type Field struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Secret      bool   `json:"secret,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// Type is registry metadata for one widget, served at /api/widgets/types.
type Type struct {
	Type        string  `json:"type"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Local       bool    `json:"local,omitempty"`
	Fields      []Field `json:"fields"`
}

// Fetcher pulls live data for one configured widget item. cfg is the item's
// config JSON — never log it, it may hold API keys.
type Fetcher interface {
	Meta() Type
	Fetch(ctx context.Context, cfg json.RawMessage) (any, error)
}

var registry = map[string]Fetcher{}

func register(f Fetcher) { registry[f.Meta().Type] = f }

// Get resolves a config.type to its fetcher.
func Get(t string) (Fetcher, bool) {
	f, ok := registry[t]
	return f, ok
}

// Types lists all registered types, sorted for a stable dialog order.
func Types() []Type {
	out := make([]Type, 0, len(registry))
	for _, f := range registry {
		out = append(out, f.Meta())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}

var client = &http.Client{Timeout: 6 * time.Second}

// getJSON fetches url and decodes the body into out. headers adds auth etc.
// The caller's config is never logged — errors carry only status codes.
func getJSON(ctx context.Context, url string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upstream unreachable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("upstream returned %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
}

// cfgStr reads a string field from a config blob.
func cfgStr(cfg json.RawMessage, key string) string {
	var m map[string]any
	if err := json.Unmarshal(cfg, &m); err != nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
