package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdvorak/dash/internal/db"
	"github.com/tdvorak/dash/internal/domain"
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
	s := newServer(zap.NewNop(), d, t.TempDir())
	// Stub the domain lookup — real RDAP/WHOIS/DNS would make tests slow and
	// network-dependent.
	s.lookup = func(_ context.Context, name string) *domain.Result {
		exp := time.Now().Add(90 * 24 * time.Hour).UTC()
		return &domain.Result{
			Name:          name,
			RegistrarName: "Test Registrar",
			ExpiryDate:    &exp,
			IPv4:          []string{"93.184.216.34"},
			NameServers:   []string{"ns1.example.com"},
			Headers:       map[string]string{},
			FaviconURL:    "https://www.google.com/s2/favicons?domain=" + name,
		}
	}
	return httptest.NewServer(s.routes())
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

func TestWidgetEndpoints(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// Types registry serves all integrations incl. the local clock.
	resp, err := http.Get(srv.URL + "/api/widgets/types")
	if err != nil {
		t.Fatalf("get types: %v", err)
	}
	var types []struct {
		Type  string `json:"type"`
		Local bool   `json:"local"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&types); err != nil {
		t.Fatalf("decode types: %v", err)
	}
	resp.Body.Close()
	got := map[string]bool{}
	for _, w := range types {
		got[w.Type] = w.Local
	}
	for _, want := range []string{"clock", "pihole", "adguard", "immich"} {
		if _, ok := got[want]; !ok {
			t.Fatalf("missing widget type %s in %v", want, got)
		}
	}
	if !got["clock"] {
		t.Fatalf("clock must be local, got %v", got)
	}

	// Widget data on a service item -> 400; missing -> 404.
	_, a := do(t, "POST", srv.URL+"/api/sections", `{"name":"A"}`)
	_, it := do(t, "POST", srv.URL+"/api/items", `{"sectionId":"`+a["id"].(string)+`","name":"Svc"}`)
	res, _ := do(t, "GET", srv.URL+"/api/widgets/"+it["id"].(string)+"/data", "")
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("service item data: %d", res.StatusCode)
	}
	res, _ = do(t, "GET", srv.URL+"/api/widgets/i_missing/data", "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing item: %d", res.StatusCode)
	}

	// Widget item with dead upstream -> 502, then cached (second call fast).
	_, w := do(t, "POST", srv.URL+"/api/items", `{"sectionId":"`+a["id"].(string)+`","kind":"widget","name":"PH","config":{"type":"pihole","endpoint":"http://127.0.0.1:1","token":"x"}}`)
	res, _ = do(t, "GET", srv.URL+"/api/widgets/"+w["id"].(string)+"/data", "")
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("dead upstream: %d", res.StatusCode)
	}
	start := time.Now()
	res, _ = do(t, "GET", srv.URL+"/api/widgets/"+w["id"].(string)+"/data", "")
	if res.StatusCode != http.StatusBadGateway || time.Since(start) > time.Second {
		t.Fatalf("expected cached 502, got %d in %v", res.StatusCode, time.Since(start))
	}
}

func TestImportExternalHomepage(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	body := `- Media:
    - Plex:
        href: http://plex.local:32400
        icon: plex
    - NoLink:
        description: widget-only entry
- Apps:
    - Sonarr:
        href: https://sonarr.example.com
        icon: https://icons.example.com/sonarr.png
`
	res, _ := do(t, "POST", srv.URL+"/api/import/external", body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("homepage import: %d", res.StatusCode)
	}
	var secs []Section
	resp, err := http.Get(srv.URL + "/api/sections")
	if err != nil {
		t.Fatalf("get sections: %v", err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&secs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(secs) != 2 {
		t.Fatalf("expected 2 sections, got %+v", secs)
	}
	media := secs[0]
	if media.Name != "Media" || len(media.Items) != 1 {
		t.Fatalf("bad Media section: %+v", media)
	}
	if media.Items[0].Name != "Plex" || len(media.Items[0].URLs) != 1 {
		t.Fatalf("bad Plex item: %+v", media.Items[0])
	}
	if media.Items[0].Icon == "" {
		t.Fatal("expected resolved CDN icon")
	}
	apps := secs[1]
	if apps.Items[0].Icon != "https://icons.example.com/sonarr.png" {
		t.Fatalf("icon passthrough failed: %+v", apps.Items[0])
	}
}

func TestImportExternalDashy(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	body := `appConfig:
  title: test
sections:
  - name: Media
    items:
      - title: Plex
        url: http://plex.local
        icon: favicon
      - title: Broken
        url: "notaurl"
`
	res, _ := do(t, "POST", srv.URL+"/api/import/external", body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("dashy import: %d", res.StatusCode)
	}
	var secs []Section
	resp, err := http.Get(srv.URL + "/api/sections")
	if err != nil {
		t.Fatalf("get sections: %v", err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&secs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(secs) != 1 || len(secs[0].Items) != 1 {
		t.Fatalf("bad sections: %+v", secs)
	}
	if secs[0].Items[0].Icon != "http://plex.local/favicon.ico" {
		t.Fatalf("favicon resolution failed: %+v", secs[0].Items[0])
	}
}

func TestImportExternalHomarr(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	body := `{"apps":[{"name":"Plex","url":"http://plex.local","icon":"hl-plex","categoryId":"c1"},{"name":"Sonarr","url":"http://sonarr.local","categoryId":"c1"}],"categories":[{"id":"c1","name":"Media"}]}`
	res, _ := do(t, "POST", srv.URL+"/api/import/external", body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("homarr import: %d", res.StatusCode)
	}
	var secs []Section
	resp, err := http.Get(srv.URL + "/api/sections")
	if err != nil {
		t.Fatalf("get sections: %v", err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&secs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(secs) != 1 || secs[0].Name != "Media" || len(secs[0].Items) != 2 {
		t.Fatalf("bad sections: %+v", secs)
	}
}

func TestImportExternalRejectsGarbage(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	res, body := do(t, "POST", srv.URL+"/api/import/external", `{"foo": 1}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %v", res.StatusCode, body)
	}
}

func TestMonitorLifecycle(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// Target that returns 200.
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer up.Close()

	res, mv := do(t, "POST", srv.URL+"/api/monitors", `{"name":"Site","type":"http","url":"`+up.URL+`","intervalS":60}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %v", res.StatusCode, mv)
	}
	id := mv["id"].(string)

	// check-now -> up, heartbeat recorded.
	res, mv = do(t, "POST", srv.URL+"/api/monitors/"+id+"/check", "")
	if res.StatusCode != http.StatusOK || mv["status"] != "up" {
		t.Fatalf("check-now: %d %v", res.StatusCode, mv)
	}
	resp, err := http.Get(srv.URL + "/api/monitors/" + id + "/heartbeats")
	if err != nil {
		t.Fatalf("heartbeats: %v", err)
	}
	var hbs []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&hbs)
	resp.Body.Close()
	if len(hbs) != 1 || hbs[0]["status"] != "up" {
		t.Fatalf("expected 1 up heartbeat, got %v", hbs)
	}

	// pause -> paused + inactive
	res, mv = do(t, "PATCH", srv.URL+"/api/monitors/"+id, `{"active":false}`)
	if res.StatusCode != http.StatusOK || mv["status"] != "paused" || mv["active"] != false {
		t.Fatalf("pause: %v", mv)
	}

	// delete
	res, _ = do(t, "DELETE", srv.URL+"/api/monitors/"+id, "")
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %d", res.StatusCode)
	}
	res, _ = do(t, "GET", srv.URL+"/api/monitors/"+id, "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", res.StatusCode)
	}
}

func TestMonitorCheckDown(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	res, mv := do(t, "POST", srv.URL+"/api/monitors", `{"name":"Dead","type":"http","url":"http://127.0.0.1:1","timeoutS":1}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create: %v", mv)
	}
	id := mv["id"].(string)
	res, mv = do(t, "POST", srv.URL+"/api/monitors/"+id+"/check", "")
	if res.StatusCode != http.StatusOK || mv["status"] != "down" {
		t.Fatalf("expected down, got %v", mv)
	}
}

func TestPushMonitor(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	res, mv := do(t, "POST", srv.URL+"/api/monitors", `{"name":"Backup","type":"push","intervalS":60}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create: %v", mv)
	}
	token, _ := mv["pushToken"].(string)
	if token == "" {
		t.Fatal("no push token issued")
	}
	res, _ = do(t, "GET", srv.URL+"/api/push/"+token+"?msg=done", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("push: %d", res.StatusCode)
	}
	res, mv = do(t, "GET", srv.URL+"/api/monitors/"+mv["id"].(string), "")
	if mv["status"] != "up" {
		t.Fatalf("expected up after push, got %v", mv["status"])
	}
	res, _ = do(t, "GET", srv.URL+"/api/push/bogus", "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("bad token: %d", res.StatusCode)
	}
}

func TestDomainLifecycle(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// Garbage name -> 400 before any lookup happens.
	res, body := do(t, "POST", srv.URL+"/api/domains", `{"name":"not a domain!!"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %v", res.StatusCode, body)
	}

	// Valid name creates even when lookups fail (unresolvable TLD).
	res, dv := do(t, "POST", srv.URL+"/api/domains", `{"name":"nonexistent.invalid","alertDaysBefore":14}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %v", res.StatusCode, dv)
	}
	if dv["tld"] != "invalid" || dv["name"] != "nonexistent.invalid" {
		t.Fatalf("bad domain row: %v", dv)
	}
	// Stubbed lookup populated registration data + computed countdown.
	if dv["registrarName"] != "Test Registrar" || dv["daysUntilExpiry"] == nil {
		t.Fatalf("lookup not applied: %v", dv)
	}
	id := dv["id"].(string)

	// Duplicate name -> 409.
	res, _ = do(t, "POST", srv.URL+"/api/domains", `{"name":"nonexistent.invalid"}`)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}

	// Patch alert threshold round-trips.
	res, dv = do(t, "PATCH", srv.URL+"/api/domains/"+id, `{"alertDaysBefore":7,"autoRenew":true}`)
	if res.StatusCode != http.StatusOK || dv["alertDaysBefore"] != float64(7) || dv["autoRenew"] != true {
		t.Fatalf("patch: %v", dv)
	}

	// Refresh endpoint answers (lookup result content is network-dependent).
	res, dv = do(t, "POST", srv.URL+"/api/domains/"+id+"/refresh", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("refresh: %d %v", res.StatusCode, dv)
	}

	// Checks history endpoint responds with a list.
	resp, err := http.Get(srv.URL + "/api/domains/" + id + "/checks")
	if err != nil {
		t.Fatalf("checks: %v", err)
	}
	var checks []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&checks)
	resp.Body.Close()

	res, _ = do(t, "DELETE", srv.URL+"/api/domains/"+id, "")
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %d", res.StatusCode)
	}
	res, _ = do(t, "GET", srv.URL+"/api/domains/"+id, "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", res.StatusCode)
	}
}

func TestDomainWidget(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	res, dv := do(t, "POST", srv.URL+"/api/domains", `{"name":"widget.invalid"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %v", res.StatusCode, dv)
	}
	_, a := do(t, "POST", srv.URL+"/api/sections", `{"name":"A"}`)
	_, w := do(t, "POST", srv.URL+"/api/items", `{"sectionId":"`+a["id"].(string)+`","kind":"widget","name":"D","config":{"type":"domain","domainId":"`+dv["id"].(string)+`"}}`)
	res, data := do(t, "GET", srv.URL+"/api/widgets/"+w["id"].(string)+"/data", "")
	inner, _ := data["data"].(map[string]any)
	if res.StatusCode != http.StatusOK || inner["name"] != "widget.invalid" {
		t.Fatalf("domain widget: %d %v", res.StatusCode, data)
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
