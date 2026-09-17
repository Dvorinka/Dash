package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tdvorak/dash/internal/db"
	"go.uber.org/zap"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "dash.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	gin.SetMode(gin.TestMode)
	return httptest.NewServer(NewRouter(zap.NewNop(), d, t.TempDir()))
}

func do(t *testing.T, method, url, body string) (*http.Response, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res, out
}

func TestSectionItemReorderFlow(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	_, a := do(t, "POST", srv.URL+"/api/sections", `{"name":"A"}`)
	_, b := do(t, "POST", srv.URL+"/api/sections", `{"name":"B"}`)
	secA, secB := a["id"].(string), b["id"].(string)

	// Move B before A: position should go below A's.
	res, mv := do(t, "POST", srv.URL+"/api/sections/reorder", `{"id":"`+secB+`","afterId":"`+secA+`"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("reorder: %v", mv)
	}
	if mv["position"].(float64) >= a["position"].(float64) {
		t.Fatalf("expected B position < A position, got %v vs %v", mv["position"], a["position"])
	}

	// Item with two urls into A.
	_, it := do(t, "POST", srv.URL+"/api/items", `{"sectionId":"`+secA+`","name":"Svc","urls":[{"url":"http://x.local","label":"local"},{"url":"https://x.dev","label":"external"}]}`)
	if res := it["id"]; res == nil {
		t.Fatalf("create item: %v", it)
	}

	// Move the item across sections.
	res, mv = do(t, "POST", srv.URL+"/api/items/reorder", `{"id":"`+it["id"].(string)+`","sectionId":"`+secB+`"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("item reorder: %v", mv)
	}

	// Board reflects the move.
	var secs []Section
	resp, err := http.Get(srv.URL + "/api/sections")
	if err != nil {
		t.Fatalf("get sections: %v", err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&secs); err != nil {
		t.Fatalf("decode board: %v", err)
	}
	if len(secs) != 2 || secs[0].Name != "B" || secs[1].Name != "A" {
		t.Fatalf("unexpected section order: %+v", secs)
	}
	if len(secs[0].Items) != 1 || secs[0].Items[0].Name != "Svc" {
		t.Fatalf("item did not move to section B: %+v", secs[0].Items)
	}
	if len(secs[0].Items[0].URLs) != 2 {
		t.Fatalf("expected 2 urls, got %+v", secs[0].Items[0].URLs)
	}
}

func TestRejectsNonHTTPURLs(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	_, a := do(t, "POST", srv.URL+"/api/sections", `{"name":"A"}`)
	res, body := do(t, "POST", srv.URL+"/api/items", `{"sectionId":"`+a["id"].(string)+`","name":"Bad","urls":[{"url":"file:///etc/passwd","label":"x"}]}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for file:// url, got %d: %v", res.StatusCode, body)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	res, _ := do(t, "PUT", srv.URL+"/api/settings", `{"theme":"light","renderer":"cards"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("put settings: %d", res.StatusCode)
	}
	resp, err := http.Get(srv.URL + "/api/settings")
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	defer resp.Body.Close()
	var got map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if got["theme"] != "light" || got["renderer"] != "cards" {
		t.Fatalf("settings mismatch: %v", got)
	}
}

func TestMidpoint(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	if got := midpoint(nil, nil); got != 1024 {
		t.Fatalf("empty midpoint: %v", got)
	}
	if got := midpoint(nil, f(1024)); got != 0 {
		t.Fatalf("before-only midpoint: %v", got)
	}
	if got := midpoint(f(1024), nil); got != 2048 {
		t.Fatalf("after-only midpoint: %v", got)
	}
	if got := midpoint(f(0), f(1024)); got != 512 {
		t.Fatalf("bounded midpoint: %v", got)
	}
}

func TestValidURL(t *testing.T) {
	for _, u := range []string{"http://a.b", "https://a.b:8443/x"} {
		if !validURL(u) {
			t.Fatalf("expected valid: %s", u)
		}
	}
	for _, u := range []string{"file:///etc/passwd", "javascript:alert(1)", "http://", "notaurl", ""} {
		if validURL(u) {
			t.Fatalf("expected invalid: %s", u)
		}
	}
}
