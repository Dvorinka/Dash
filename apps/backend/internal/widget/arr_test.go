package widget

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeArr serves the *arr v3 endpoints plus the autobrr/bazarr/overseerr
// shapes, all behind the auth headers each family expects.
func fakeArr(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "k" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`[{"type":"warning","source":"DownloadClientCheck","message":"qBittorrent unreachable"}]`))
	})
	mux.HandleFunc("/api/v3/system/status", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"4.0.1.929"}`))
	})
	mux.HandleFunc("/api/v3/queue", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"totalRecords":7,"records":[]}`))
	})
	mux.HandleFunc("/api/v3/update", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"installed":true,"installable":true}]`))
	})
	// Prowlarr/lidarr/readarr speak v1 — same handlers, different prefix.
	for _, p := range []string{"/api/v1/health", "/api/v1/system/status", "/api/v1/update"} {
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/health":
				_, _ = w.Write([]byte(`[]`))
			case "/api/v1/system/status":
				_, _ = w.Write([]byte(`{"version":"1.0.0"}`))
			case "/api/v1/update":
				_, _ = w.Write([]byte(`[]`))
			}
		})
	}
	mux.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"1.13.6","updateAvailable":false}`))
	})
	mux.HandleFunc("/api/v1/indexer", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1},{"id":2},{"id":3}]`))
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Token") != "k" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"version":"1.42.0"}`))
	})
	mux.HandleFunc("/api/release/stats", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"total_count":1234,"push_error_count":2}`))
	})
	mux.HandleFunc("/api/irc", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"name":"sorcery","healthy":false,"enabled":true}]`))
	})
	mux.HandleFunc("/api/updates/latest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "k" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"data":{"bazarr_version":"1.5.1"}}`))
	})
	mux.HandleFunc("/api/system/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"object":"Radarr","issue":"cannot connect"}]}`))
	})
	mux.HandleFunc("/api/v1/request", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"pageInfo":{"results":2},"results":[{"status":1},{"status":4}]}`))
	})
	return httptest.NewServer(mux)
}

func cfgFor(endpoint string) json.RawMessage {
	cfg, _ := json.Marshal(map[string]any{"endpoint": endpoint, "apiKey": "k"})
	return cfg
}

func fetchSvc(t *testing.T, f Fetcher, cfg json.RawMessage) map[string]any {
	t.Helper()
	got, err := f.Fetch(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := got.(*svcData)
	if !ok {
		t.Fatalf("unexpected payload type %T", got)
	}
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}

func TestArrWidgetFetch(t *testing.T) {
	srv := fakeArr(t)
	defer srv.Close()

	got := fetchSvc(t, arrWidget{typ: "sonarr", apiV: "v3", queue: true}, cfgFor(srv.URL))
	if got["status"] != "warning" || got["issues"].(float64) != 1 {
		t.Fatalf("expected warning with 1 issue, got %v", got)
	}
	if got["version"] != "4.0.1.929" {
		t.Fatalf("version: %v", got["version"])
	}
	if got["queueTotal"].(float64) != 7 {
		t.Fatalf("queueTotal: %v", got["queueTotal"])
	}
	if got["updateAvailable"] == true {
		t.Fatal("installed+installable update must not count as available")
	}
}

func TestProwlarrWidgetFetch(t *testing.T) {
	srv := fakeArr(t)
	defer srv.Close()

	got := fetchSvc(t, arrWidget{typ: "prowlarr", apiV: "v1", indexer: true}, cfgFor(srv.URL))
	if got["indexers"].(float64) != 3 {
		t.Fatalf("indexers: %v", got)
	}
	if _, hasQueue := got["queueTotal"]; hasQueue {
		t.Fatal("prowlarr must not report a queue")
	}
}

func TestAutobrrWidgetFetch(t *testing.T) {
	srv := fakeArr(t)
	defer srv.Close()

	got := fetchSvc(t, autobrrWidget{}, cfgFor(srv.URL))
	if got["status"] != "warning" {
		t.Fatalf("expected warning (unhealthy IRC + push errors), got %v", got)
	}
	if got["version"] != "1.42.0" || got["releases"].(float64) != 1234 {
		t.Fatalf("version/releases: %v", got)
	}
	if got["updateAvailable"] == true {
		t.Fatal("204 from /api/updates/latest must mean no update")
	}
}

func TestBazarrWidgetFetch(t *testing.T) {
	srv := fakeArr(t)
	defer srv.Close()

	got := fetchSvc(t, bazarrWidget{}, cfgFor(srv.URL))
	if got["version"] != "1.5.1" || got["status"] != "warning" {
		t.Fatalf("unexpected payload: %v", got)
	}
}

func TestReqWidgetFetch(t *testing.T) {
	srv := fakeArr(t)
	defer srv.Close()

	got := fetchSvc(t, reqWidget{typ: "overseerr"}, cfgFor(srv.URL))
	if got["pendingRequests"].(float64) != 1 || got["status"] != "online" {
		t.Fatalf("unexpected payload: %v", got)
	}
}

func TestServiceWidgetsRequireConfig(t *testing.T) {
	for _, f := range []Fetcher{
		arrWidget{apiV: "v3", queue: true},
		autobrrWidget{},
		bazarrWidget{},
		reqWidget{},
	} {
		if _, err := f.Fetch(context.Background(), json.RawMessage(`{}`)); err == nil {
			t.Errorf("%s: expected error for empty config", f.Meta().Type)
		}
	}
}
