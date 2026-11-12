package widget

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConnectWidgets(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil || r.Form.Get("username") != "admin" || r.Form.Get("password") != "secret" {
			http.Error(w, "no", http.StatusForbidden)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "abc"})
		_, _ = w.Write([]byte("Ok."))
	})
	mux.HandleFunc("/api/v2/transfer/info", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("SID"); err != nil || c.Value != "abc" {
			http.Error(w, "no", http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(`{"dl_info_speed":1500000,"ul_info_speed":4096}`))
	})
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"name":"a","state":"downloading"},{"name":"b","state":"pausedDL"},{"name":"bad","state":"error"}]`))
	})
	mux.HandleFunc("/api2/json/cluster/resources", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "PVEAPIToken=root@pam!dash=secret" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"type":"node","status":"offline","name":"pve2"},{"type":"qemu","status":"running"},{"type":"lxc","status":"stopped"}]}`))
	})
	mux.HandleFunc("/api/endpoints", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "pkey" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`[{"Id":1,"Name":"local","Snapshots":[{"DockerSnapshot":{"RunningContainerCount":4,"ContainerCount":5}}]}]`))
	})
	mux.HandleFunc("/api/tokens", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"token":"npm-tok"}`))
	})
	mux.HandleFunc("/api/nginx/proxy-hosts", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer npm-tok" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`[{"enabled":true},{"enabled":false},{"enabled":true}]`))
	})
	mux.HandleFunc("/api/states", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer hatok" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`[{"entity_id":"light.a","state":"on"},{"entity_id":"sensor.b","state":"unavailable"}]`))
	})
	mux.HandleFunc("/rest/system/version", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "st" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"version":"1.27.2"}`))
	})
	mux.HandleFunc("/rest/system/connections", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"connections":{"A":{"connected":true},"B":{"connected":false}}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	q := fetchSvc(t, qbittorrentWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL, "username": "admin", "apiKey": "secret"}))
	if q["status"] != "warning" || q["name"] != "1/3 active" || q["issues"] != float64(1) || q["warning"] != "bad" || q["speed"] != "dl 1.4 MB/s · ul 4 KB/s" {
		t.Fatalf("qbit: %v", q)
	}

	p := fetchSvc(t, proxmoxWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL, "apiKey": "root@pam!dash=secret"}))
	if p["up"] != float64(1) || p["total"] != float64(2) || p["status"] != "warning" || p["warning"] != "pve2" {
		t.Fatalf("proxmox: %v", p)
	}

	o := fetchSvc(t, portainerWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL, "apiKey": "pkey"}))
	if o["name"] != "local" || o["up"] != float64(4) || o["total"] != float64(5) {
		t.Fatalf("portainer: %v", o)
	}

	n := fetchSvc(t, npmWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL, "username": "a@b.c", "apiKey": "pw"}))
	if n["up"] != float64(2) || n["total"] != float64(3) {
		t.Fatalf("npm: %v", n)
	}

	h := fetchSvc(t, homeAssistantWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL, "apiKey": "hatok"}))
	if h["up"] != float64(1) || h["total"] != float64(2) || h["status"] != "warning" || h["warning"] != "sensor.b" {
		t.Fatalf("ha: %v", h)
	}

	s := fetchSvc(t, syncthingWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL, "apiKey": "st"}))
	if s["version"] != "1.27.2" || s["up"] != float64(1) || s["total"] != float64(2) || s["status"] != "online" {
		t.Fatalf("syncthing: %v", s)
	}
}

func TestConnectRequiresConfig(t *testing.T) {
	if _, err := (qbittorrentWidget{}).Fetch(context.Background(), []byte(`{}`)); err == nil {
		t.Fatal("expected missing endpoint to fail")
	}
}

func TestPortainerContainerFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/endpoints", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"Id":7,"Name":"bare","Snapshots":[]}]`))
	})
	mux.HandleFunc("/api/endpoints/7/docker/containers/json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"State":"running"},{"State":"exited"}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	d := fetchSvc(t, portainerWidget{}, svcCfg(t, map[string]string{"endpoint": srv.URL, "apiKey": "k"}))
	if d["up"] != float64(1) || d["total"] != float64(2) || d["name"] != "bare" {
		t.Fatalf("fallback: %v", d)
	}
}
