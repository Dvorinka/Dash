package widget

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// JSON status: GET a URL, extract one value by dot/bracket path
// ("data.status", "items[0].name"), render it in a tile. Covers the
// "is API X alive / what does it report" case without per-service widgets.
type jsonpath struct{}

func init() { Register(jsonpath{}) }

func (jsonpath) Meta() Type {
	return Type{
		Type:        "jsonpath",
		Name:        "JSON value",
		Description: "Extract one value from a JSON endpoint",
		Fields: []Field{
			{Key: "url", Label: "URL", Placeholder: "https://api…/status", Required: true},
			{Key: "path", Label: "Path", Placeholder: "data.status", Required: true},
			{Key: "header", Label: "Header", Placeholder: "Authorization: Bearer …"},
			{Key: "label", Label: "Label", Placeholder: "Status"},
		},
	}
}

func (jsonpath) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	url := cfgStr(cfg, "url")
	path := cfgStr(cfg, "path")
	if url == "" || path == "" {
		return nil, fmt.Errorf("url and path required")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("url must be http(s)")
	}
	var headers map[string]string
	if h := cfgStr(cfg, "header"); h != "" {
		if k, v, ok := strings.Cut(h, ":"); ok {
			headers = map[string]string{strings.TrimSpace(k): strings.TrimSpace(v)}
		}
	}
	var doc any
	if err := getJSON(ctx, url, headers, &doc); err != nil {
		return nil, err
	}
	v, ok := jsonPath(doc, path)
	if !ok {
		return nil, fmt.Errorf("path %q not found", path)
	}
	return map[string]any{"value": v, "label": cfgStr(cfg, "label")}, nil
}

// jsonPath walks doc along a path like "a.b[0].c". Segments split on dots;
// a trailing [n] indexes into arrays. Returns false on any miss.
func jsonPath(doc any, path string) (any, bool) {
	cur := doc
	for _, seg := range strings.Split(path, ".") {
		name, rest, _ := strings.Cut(seg, "[")
		if name != "" {
			m, ok := cur.(map[string]any)
			if !ok {
				return nil, false
			}
			cur, ok = m[name]
			if !ok {
				return nil, false
			}
		}
		for rest != "" {
			idxStr, tail, _ := strings.Cut(rest, "]")
			i, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, false
			}
			arr, ok := cur.([]any)
			if !ok || i < 0 || i >= len(arr) {
				return nil, false
			}
			cur = arr[i]
			rest = strings.TrimPrefix(tail, "[")
		}
	}
	return cur, true
}
