package widget

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Immich REST API: GET /api/statistics with an x-api-key header.
type immich struct{}

func init() { Register(immich{}) }

func (immich) Meta() Type {
	return Type{
		Type:        "immich",
		Name:        "Immich",
		Description: "Library size and photo count",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://immich.local:2283", Required: true},
			{Key: "apiKey", Label: "API key", Secret: true, Required: true},
		},
	}
}

type immichData struct {
	Photos int   `json:"photos"`
	Videos int   `json:"videos"`
	UsageB int64 `json:"usageBytes"`
}

func (immich) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	key := cfgStr(cfg, "apiKey")
	if endpoint == "" || key == "" {
		return nil, fmt.Errorf("endpoint and apiKey required")
	}
	var raw struct {
		Images int   `json:"images"`
		Videos int   `json:"videos"`
		Usage  int64 `json:"usage"`
	}
	if err := getJSON(ctx, endpoint+"/api/statistics",
		map[string]string{"x-api-key": key}, &raw); err != nil {
		return nil, err
	}
	return immichData{
		Photos: raw.Images,
		Videos: raw.Videos,
		UsageB: raw.Usage,
	}, nil
}
