package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatusPingUsesGET(t *testing.T) {
	var method string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		if r.Method != http.MethodGet {
			http.Error(w, "no", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	sc := newStatusCache(time.Minute)
	e := sc.ping(srv.URL)
	if method != http.MethodGet {
		t.Fatalf("probe method %s", method)
	}
	if e.Status != "up" {
		t.Fatalf("status %s", e.Status)
	}
	if e.LatencyMs < 0 {
		t.Fatalf("latency %d", e.LatencyMs)
	}
}

func TestStatusPingDownOn5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)

	e := newStatusCache(time.Minute).ping(srv.URL)
	if e.Status != "down" {
		t.Fatalf("status %s", e.Status)
	}
}
