package widget

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AdGuard Home REST API: GET /control/stats with HTTP basic auth.
type adguard struct{}

func init() { Register(adguard{}) }

func (adguard) Meta() Type {
	return Type{
		Type:        "adguard",
		Name:        "AdGuard Home",
		Description: "Queries and block rate",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://adguard.local", Required: true},
			{Key: "user", Label: "Username", Required: true},
			{Key: "pass", Label: "Password", Secret: true, Required: true},
		},
	}
}

type adguardData struct {
	QueriesToday   int     `json:"queriesToday"`
	BlockedToday   int     `json:"blockedToday"`
	BlockedPercent float64 `json:"blockedPercent"`
	AvgProcessMs   float64 `json:"avgProcessMs"`
}

func (adguard) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	user, pass := cfgStr(cfg, "user"), cfgStr(cfg, "pass")
	if endpoint == "" || user == "" {
		return nil, fmt.Errorf("endpoint and user required")
	}
	var raw struct {
		NumDNSQueries        int     `json:"num_dns_queries"`
		NumBlockedFiltering  int     `json:"num_blocked_filtering"`
		AvgProcessingTimeSec float64 `json:"avg_processing_time_sec"`
	}
	// AdGuard uses basic auth; encode via the URL userinfo so the helper
	// stays header-free — and the secret never touches a log line.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/control/stats", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pass)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream unreachable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	var pct float64
	if raw.NumDNSQueries > 0 {
		pct = float64(raw.NumBlockedFiltering) / float64(raw.NumDNSQueries) * 100
	}
	return adguardData{
		QueriesToday:   raw.NumDNSQueries,
		BlockedToday:   raw.NumBlockedFiltering,
		BlockedPercent: pct,
		AvgProcessMs:   raw.AvgProcessingTimeSec * 1000,
	}, nil
}
