package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tdvorak/dash/internal/domain"
)

// ---------- transports ----------

func TestNotifyTransports(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// One receiver records body + headers + query per request.
	type hit struct {
		body   map[string]any
		raw    string
		title  string
		path   string
	}
	got := make(chan hit, 8)
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]any
		buf, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(buf, &m)
		got <- hit{body: m, raw: string(buf), title: r.Header.Get("Title"), path: r.URL.RequestURI()}
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	put := func(kv string) {
		if res, _ := do(t, "PUT", srv.URL+"/api/settings", kv); res.StatusCode != http.StatusOK {
			t.Fatalf("put %s: %d", kv, res.StatusCode)
		}
	}

	// slack + discord + gotify + ntfy all configured; test fires each.
	put(`{"notify_slack":{"url":"` + hook.URL + `/slack"}}`)
	put(`{"notify_discord":{"url":"` + hook.URL + `/discord"}}`)
	put(`{"notify_gotify":{"server":"` + hook.URL + `","token":"gtok"}}`)
	put(`{"notify_ntfy":{"server":"` + hook.URL + `","topic":"alerts"}}`)

	res, out := do(t, "POST", srv.URL+"/api/notify/test", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("notify test: %d", res.StatusCode)
	}
	results := out["results"].(map[string]any)
	for _, name := range []string{"slack", "discord", "gotify", "ntfy"} {
		if results[name] != "ok" {
			t.Fatalf("%s: %v", name, results[name])
		}
	}

	seen := map[string]hit{}
	for i := 0; i < 4; i++ {
		select {
		case h := <-got:
			seen[h.path] = h
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for transports")
		}
	}
	if seen["/slack"].body["text"] == "" {
		t.Fatalf("slack payload missing text: %v", seen["/slack"].body)
	}
	if seen["/discord"].body["content"] == "" {
		t.Fatalf("discord payload missing content: %v", seen["/discord"].body)
	}
	if !strings.HasPrefix(seen["/message?token=gtok"].path, "/message?token=") {
		t.Fatalf("gotify hit missing: %v", seen)
	}
	if h := seen["/alerts"]; h.raw != "Dash test notification" || h.title != "dash" {
		t.Fatalf("ntfy payload wrong: %q title=%q", h.raw, h.title)
	}

	// Legacy string webhook still works alongside presets.
	put(`{"notify_webhook":"` + hook.URL + `/legacy"}`)
	res, _ = do(t, "POST", srv.URL+"/api/notify/test", `{"transport":"webhook"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("webhook test: %d", res.StatusCode)
	}
	select {
	case h := <-got:
		if h.path != "/legacy" || h.body["event"] != "test" {
			t.Fatalf("legacy webhook payload: %v", h.body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("legacy webhook never fired")
	}

	// Named unconfigured transport -> 400.
	res, _ = do(t, "POST", srv.URL+"/api/notify/test", `{"transport":"smtp"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unconfigured smtp, got %d", res.StatusCode)
	}
}

// fakeSMTP is the smallest lawful SMTP server: EHLO/MAIL/RCPT/DATA/QUIT.
func fakeSMTP(t *testing.T) (addr string, gotMsg <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	msgs := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		fmt.Fprint(conn, "220 fake\r\n")
		var data strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if inData {
				if line == "." {
					inData = false
					fmt.Fprint(conn, "250 queued\r\n")
					continue
				}
				data.WriteString(line + "\n")
				continue
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				fmt.Fprint(conn, "250-fake\r\n250 OK\r\n")
			case strings.HasPrefix(line, "MAIL"), strings.HasPrefix(line, "RCPT"):
				fmt.Fprint(conn, "250 OK\r\n")
			case strings.HasPrefix(line, "DATA"):
				fmt.Fprint(conn, "354 go\r\n")
				inData = true
			case strings.HasPrefix(line, "QUIT"):
				fmt.Fprint(conn, "221 bye\r\n")
				msgs <- data.String()
				return
			default:
				fmt.Fprint(conn, "250 OK\r\n")
			}
		}
	}()
	return ln.Addr().String(), msgs
}

func TestNotifySMTP(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	addr, gotMsg := fakeSMTP(t)
	host, port, _ := net.SplitHostPort(addr)
	res, _ := do(t, "PUT", srv.URL+"/api/settings",
		`{"notify_smtp":{"host":"`+host+`","port":"`+port+`","from":"dash@local","to":"me@local"}}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("put smtp: %d", res.StatusCode)
	}
	res, out := do(t, "POST", srv.URL+"/api/notify/test", `{"transport":"smtp"}`)
	if res.StatusCode != http.StatusOK || out["results"].(map[string]any)["smtp"] != "ok" {
		t.Fatalf("smtp test: %d %v", res.StatusCode, out)
	}
	select {
	case msg := <-gotMsg:
		if !strings.Contains(msg, "Subject: [Dash] Dash test notification") {
			t.Fatalf("smtp message missing subject: %q", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("smtp server never received the message")
	}
}

// ---------- alert rules ----------

func TestAlertRulesConsecutiveAndMute(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	got := make(chan map[string]any, 8)
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]any
		_ = json.NewDecoder(r.Body).Decode(&m)
		got <- m
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()
	do(t, "PUT", srv.URL+"/api/settings", `{"notify_webhook":"`+hook.URL+`"}`)

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer up.Close()
	down := `{"url":"http://127.0.0.1:1","timeoutS":1}`

	// Monitor requiring 2 consecutive failures before going down.
	res, m := do(t, "POST", srv.URL+"/api/monitors",
		`{"name":"Flap","type":"http","url":"`+up.URL+`","timeoutS":1,"alerts":{"consecutiveFailures":2}}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create: %v", m)
	}
	id := m["id"].(string)
	do(t, "POST", srv.URL+"/api/monitors/"+id+"/check", "") // -> up

	do(t, "PATCH", srv.URL+"/api/monitors/"+id, down)
	do(t, "POST", srv.URL+"/api/monitors/"+id+"/check", "") // 1st failure: hold
	_, mv := do(t, "GET", srv.URL+"/api/monitors/"+id, "")
	if mv["status"] != "up" {
		t.Fatalf("expected held status up after 1 failure, got %v", mv["status"])
	}
	do(t, "POST", srv.URL+"/api/monitors/"+id+"/check", "") // 2nd failure: down
	_, mv = do(t, "GET", srv.URL+"/api/monitors/"+id, "")
	if mv["status"] != "down" {
		t.Fatalf("expected down after 2 failures, got %v", mv["status"])
	}
	select {
	case body := <-got:
		if body["event"] != "monitor.down" {
			t.Fatalf("expected monitor.down, got %v", body["event"])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("monitor.down never fired after threshold")
	}

	// Muted monitor: status still flips, no notification.
	res, m2 := do(t, "POST", srv.URL+"/api/monitors",
		`{"name":"Quiet","type":"http","url":"`+up.URL+`","timeoutS":1,"alerts":{"mute":true}}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create muted: %v", m2)
	}
	id2 := m2["id"].(string)
	do(t, "POST", srv.URL+"/api/monitors/"+id2+"/check", "")
	do(t, "PATCH", srv.URL+"/api/monitors/"+id2, down)
	do(t, "POST", srv.URL+"/api/monitors/"+id2+"/check", "")
	_, mv = do(t, "GET", srv.URL+"/api/monitors/"+id2, "")
	if mv["status"] != "down" {
		t.Fatalf("muted monitor should still go down, got %v", mv["status"])
	}
	select {
	case body := <-got:
		if body["name"] == "Quiet" {
			t.Fatal("muted monitor still notified")
		}
	case <-time.After(1500 * time.Millisecond):
	}
}

func TestAlertRulesBadJSON(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	res, _ := do(t, "POST", srv.URL+"/api/monitors",
		`{"name":"X","type":"http","url":"http://127.0.0.1:1","alerts":"nope"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid alerts, got %d", res.StatusCode)
	}
}

// ---------- subdomains ----------

func TestSubdomainDiscovery(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// Stub the network seams: CT returns candidates, resolver answers a few.
	oldCT, oldRes := domain.CTNames, domain.ResolveIPs
	domain.CTNames = func(_ context.Context, d string) ([]string, error) {
		return []string{"api." + d + "\nwww." + d, "*." + d, "other.com"}, nil
	}
	domain.ResolveIPs = func(_ context.Context, name string) ([]string, error) {
		if strings.HasPrefix(name, "api.") {
			return []string{"10.0.0.1"}, nil
		}
		return []string{"93.184.216.34"}, nil
	}
	t.Cleanup(func() { domain.CTNames, domain.ResolveIPs = oldCT, oldRes })

	res, d := do(t, "POST", srv.URL+"/api/domains", `{"name":"example.com"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create domain: %v", d)
	}
	id := d["id"].(string)

	// First refresh ran discovery inline (sub_checked_at was NULL).
	res, subs := do(t, "GET", srv.URL+"/api/domains/"+id+"/subdomains", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list subdomains: %d", res.StatusCode)
	}
	list, _ := subs["subdomains"].([]any) // response is a bare array — check below
	_ = list
	// do() decodes objects only; re-fetch raw for the array.
	raw, err := http.Get(srv.URL + "/api/domains/" + id + "/subdomains")
	if err != nil {
		t.Fatal(err)
	}
	var arr []map[string]any
	_ = json.NewDecoder(raw.Body).Decode(&arr)
	raw.Body.Close()
	names := map[string][]string{}
	for _, s := range arr {
		ips := []string{}
		for _, ip := range s["ips"].([]any) {
			ips = append(ips, ip.(string))
		}
		names[s["name"].(string)] = ips
	}
	if len(names) != 3 || len(names["api.example.com"]) != 1 || names["api.example.com"][0] != "10.0.0.1" {
		t.Fatalf("unexpected subdomains: %v", names)
	}
	if _, bad := names["other.com"]; bad {
		t.Fatal("foreign name leaked into subdomains")
	}
}
