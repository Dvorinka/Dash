package widgets

import (
	"encoding/json"
	"testing"
)

func TestNormalizePiHoleClassicSummary(t *testing.T) {
	got, err := normalizePiHole(map[string]any{
		"ads_blocked_today":    float64(25),
		"dns_queries_today":    float64(100),
		"ads_percentage_today": float64(25),
		"status":               "enabled",
	})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["blockedCount"] != float64(25) || payload["queryCount"] != float64(100) {
		t.Fatalf("unexpected payload: %v", payload)
	}
}

func TestNormalizePiHoleV6Summary(t *testing.T) {
	got, err := normalizePiHole(map[string]any{
		"queries": map[string]any{
			"blocked":         float64(30),
			"total":           float64(120),
			"percent_blocked": float64(25),
		},
		"status": "enabled",
	})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["percentBlocked"] != float64(25) {
		t.Fatalf("unexpected payload: %v", payload)
	}
}
