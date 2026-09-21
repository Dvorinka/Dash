package widget

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Pi-hole v5 HTTP API: /admin/api.php?summaryRaw&auth=<token>.
// jarvis: v5 only for now — v6 (/api/...) lands when someone runs it.
type pihole struct{}

func init() { Register(pihole{}) }

func (pihole) Meta() Type {
	return Type{
		Type:        "pihole",
		Name:        "Pi-hole",
		Description: "Queries and block rate",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://pi.hole", Required: true},
			{Key: "token", Label: "API token", Secret: true, Placeholder: "web password / api token"},
		},
	}
}

type piholeData struct {
	QueriesToday   int     `json:"queriesToday"`
	BlockedToday   int     `json:"blockedToday"`
	BlockedPercent float64 `json:"blockedPercent"`
	GravitySize    int     `json:"gravitySize"`
	Status         string  `json:"status"`
}

func (pihole) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint required")
	}
	u := endpoint + "/admin/api.php?summaryRaw"
	if tok := cfgStr(cfg, "token"); tok != "" {
		u += "&auth=" + tok
	}
	var raw struct {
		DNSQueriesToday int     `json:"dns_queries_today"`
		AdsBlockedToday int     `json:"ads_blocked_today"`
		AdsPercentToday float64 `json:"ads_percentage_today"`
		DomainsBlocked  int     `json:"domains_being_blocked"`
		Status          string  `json:"status"`
	}
	if err := getJSON(ctx, u, nil, &raw); err != nil {
		return nil, err
	}
	return piholeData{
		QueriesToday:   raw.DNSQueriesToday,
		BlockedToday:   raw.AdsBlockedToday,
		BlockedPercent: raw.AdsPercentToday,
		GravitySize:    raw.DomainsBlocked,
		Status:         raw.Status,
	}, nil
}
