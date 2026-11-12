package widget

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func svcCfg(t *testing.T, m map[string]string) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestMediaWidgets(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/System/Info", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != `MediaBrowser Token="k"` {
			t.Errorf("bad auth header: %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"ServerName":"Jelly","Version":"10.9.11"}`))
	})
	mux.HandleFunc("/Sessions", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"Id":"a"},{"Id":"b"}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	w := mediaWidget{typ: "jellyfin", name: "Jellyfin", verPath: "/System/Info", sesPath: "/Sessions",
		headers: func(k string) map[string]string {
			return map[string]string{"Authorization": `MediaBrowser Token="` + k + `"`}
		}}
	d := fetchSvc(t, w, svcCfg(t, map[string]string{"endpoint": srv.URL, "apiKey": "k"}))
	if d["status"] != "online" || d["version"] != "10.9.11" || d["name"] != "Jelly" || d["streams"] != float64(2) {
		t.Fatalf("unexpected: %v", d)
	}
}

func TestSabnzbdWidget(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("mode") {
		case "version":
			_, _ = w.Write([]byte(`{"version":"4.5.3"}`))
		case "queue":
			_, _ = w.Write([]byte(`{"queue":{"noofslots":3,"speed":"1.2 M","paused":false}}`))
		default:
			w.WriteHeader(400)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	d := fetchSvc(t, sabnzbdWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL, "apiKey": "k"}))
	if d["version"] != "4.5.3" || d["queueTotal"] != float64(3) || d["speed"] != "1.2 M" {
		t.Fatalf("unexpected: %v", d)
	}
}

func TestNzbgetWidget(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/jsonrpc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST")
		}
		if r.Header.Get("Authorization") == "" {
			t.Errorf("missing basic auth")
		}
		var req struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		switch req.Method {
		case "status":
			_, _ = w.Write([]byte(`{"result":{"DownloadPaused":true,"PostPaused":false,"ScanPaused":false,"QuotaReached":false,"DownloadRate":2097152}}`))
		case "version":
			_, _ = w.Write([]byte(`{"result":"2.4.0"}`))
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	d := fetchSvc(t, nzbgetWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL, "apiKey": "pw"}))
	if d["status"] != "warning" || d["version"] != "2.4.0" || d["speed"] == "" {
		t.Fatalf("unexpected: %v", d)
	}
}

func TestQuiWidget(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/api/instances", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1,"connected":true},{"id":2,"connected":false}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	d := fetchSvc(t, quiWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL, "apiKey": "k"}))
	if d["status"] != "warning" || d["total"] != float64(2) || d["up"] != float64(1) || d["issues"] != float64(1) {
		t.Fatalf("unexpected: %v", d)
	}
}

func TestKumaWidget(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`# HELP monitor_status x
monitor_status{monitor_id="1",monitor_name="API"} 1
monitor_status{monitor_id="2",monitor_name="DB"} 0
monitor_status{monitor_id="3",monitor_name="Web"} 1
`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	d := fetchSvc(t, kumaWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL, "apiKey": "k"}))
	if d["status"] != "warning" || d["total"] != float64(3) || d["up"] != float64(2) || d["issues"] != float64(1) || d["warning"] != "DB" {
		t.Fatalf("unexpected: %v", d)
	}
}

func TestTraefikWidget(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"3.3.2"}`))
	})
	mux.HandleFunc("/api/overview", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"http":{"routers":{"total":12,"warnings":2,"errors":0}}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	d := fetchSvc(t, traefikWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL}))
	if d["status"] != "warning" || d["version"] != "3.3.2" || d["total"] != float64(12) || d["issues"] != float64(2) {
		t.Fatalf("unexpected: %v", d)
	}
}

func TestMaintainerrWidget(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/app/status", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"1.2.3","updateAvailable":true}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	d := fetchSvc(t, maintainerrWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL}))
	if d["status"] != "online" || d["version"] != "1.2.3" || d["updateAvailable"] != true {
		t.Fatalf("unexpected: %v", d)
	}
}
