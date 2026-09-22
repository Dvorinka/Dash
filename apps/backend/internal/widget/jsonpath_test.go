package widget

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONPathWalk(t *testing.T) {
	doc := map[string]any{
		"data": map[string]any{
			"status": "ok",
			"items":  []any{map[string]any{"name": "first"}, "second"},
			"nested": []any{[]any{float64(42)}},
		},
	}
	cases := []struct {
		path string
		want any
		ok   bool
	}{
		{"data.status", "ok", true},
		{"data.items[0].name", "first", true},
		{"data.items[1]", "second", true},
		{"data.nested[0][0]", float64(42), true},
		{"data.missing", nil, false},
		{"data.items[9]", nil, false},
		{"data.items[-1]", nil, false},
		{"data.status.x", nil, false},
		{"data.items[x]", nil, false},
		{"data.status[0]", nil, false},
	}
	for _, tc := range cases {
		got, ok := jsonPath(doc, tc.path)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("path %q: got %v,%v want %v,%v", tc.path, got, ok, tc.want, tc.ok)
		}
	}
}

func TestJSONPathFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"status":"degraded"}}`))
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(map[string]any{
		"url":    srv.URL,
		"path":   "data.status",
		"header": "Authorization: Bearer tok",
		"label":  "State",
	})
	got, err := (jsonpath{}).Fetch(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	if m["value"] != "degraded" || m["label"] != "State" {
		t.Fatalf("unexpected payload: %v", m)
	}
}

func TestJSONPathFetchValidation(t *testing.T) {
	for _, cfg := range []string{
		`{"url":"","path":"a"}`,
		`{"url":"file:///etc/passwd","path":"a"}`,
		`{"url":"ftp://x","path":"a"}`,
	} {
		if _, err := (jsonpath{}).Fetch(context.Background(), json.RawMessage(cfg)); err == nil {
			t.Errorf("expected error for %s", cfg)
		}
	}
}
