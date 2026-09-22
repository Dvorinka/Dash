package api

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdvorak/dash/internal/db"
	"go.uber.org/zap"
)

// authServer returns the bare server (no httptest) so tests can plant
// sessions and inspect cookies directly.
func authServer(t *testing.T) *Server {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "dash.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	gin.SetMode(gin.TestMode)
	// Login rate limits are keyed by IP in a package-global map — reset so
	// tests don't count each other's attempts.
	loginMu.Lock()
	loginHits = map[string][]time.Time{}
	loginMu.Unlock()
	return newServer(zap.NewNop(), d, t.TempDir())
}

func req(t *testing.T, s *Server, method, path, body, cookie string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		r.Header.Set("Cookie", sessionCookie+"="+cookie)
	}
	s.routes().ServeHTTP(w, r)
	return w
}

func sessionCookieOf(w *httptest.ResponseRecorder) string {
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			return c.Value
		}
	}
	return ""
}

func enableAuth(t *testing.T, s *Server) {
	t.Helper()
	if w := req(t, s, "PUT", "/api/settings", `{"auth_enabled":true}`, ""); w.Code != 200 {
		t.Fatalf("enable auth: %d", w.Code)
	}
}

func TestAuthDisabledByDefault(t *testing.T) {
	s := authServer(t)
	if w := req(t, s, "GET", "/api/sections", "", ""); w.Code != 200 {
		t.Fatalf("sections open when auth off: %d", w.Code)
	}
	var out map[string]any
	w := req(t, s, "GET", "/api/auth/session", "", "")
	_ = json.NewDecoder(w.Body).Decode(&out)
	if out["enabled"] != false {
		t.Fatalf("auth should report disabled: %v", out)
	}
}

func TestAuthLifecycle(t *testing.T) {
	s := authServer(t)
	enableAuth(t, s)

	// Everything gates now.
	if w := req(t, s, "GET", "/api/sections", "", ""); w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	// Session endpoint stays public and reports setup mode.
	var sess map[string]any
	w := req(t, s, "GET", "/api/auth/session", "", "")
	_ = json.NewDecoder(w.Body).Decode(&sess)
	if sess["enabled"] != true || sess["setup"] != true || sess["authed"] != false {
		t.Fatalf("bad session payload: %v", sess)
	}
	// Public surfaces stay open even gated.
	for _, p := range []string{"/api/healthz", "/api/metrics", "/api/badge/uptime/x.svg"} {
		if w := req(t, s, "GET", p, "", ""); w.Code == 401 {
			t.Fatalf("public path %s gated", p)
		}
	}

	// Setup creates the first admin and returns a session cookie.
	w = req(t, s, "POST", "/api/auth/setup", `{"username":"admin","password":"correct-horse"}`, "")
	if w.Code != 201 {
		t.Fatalf("setup: %d body=%s", w.Code, w.Body)
	}
	cookie := sessionCookieOf(w)
	if cookie == "" {
		t.Fatal("setup did not set a session cookie")
	}
	// Second setup blocked.
	if w := req(t, s, "POST", "/api/auth/setup", `{"username":"x","password":"another-pass"}`, ""); w.Code != 409 {
		t.Fatalf("second setup should 409, got %d", w.Code)
	}

	// Authed request with the cookie.
	if w := req(t, s, "GET", "/api/sections", "", cookie); w.Code != 200 {
		t.Fatalf("authed request: %d", w.Code)
	}

	// Logout kills the session.
	req(t, s, "POST", "/api/auth/logout", "", cookie)
	if w := req(t, s, "GET", "/api/sections", "", cookie); w.Code != 401 {
		t.Fatalf("expected 401 after logout, got %d", w.Code)
	}

	// Login wrong password -> 401; right -> 200 + fresh cookie.
	if w := req(t, s, "POST", "/api/auth/login", `{"username":"admin","password":"wrong-password"}`, ""); w.Code != 401 {
		t.Fatalf("bad login should 401, got %d", w.Code)
	}
	w = req(t, s, "POST", "/api/auth/login", `{"username":"admin","password":"correct-horse"}`, "")
	if w.Code != 200 {
		t.Fatalf("login: %d", w.Code)
	}
	if sessionCookieOf(w) == "" {
		t.Fatal("login set no cookie")
	}
}

func TestAuthLoginRateLimit(t *testing.T) {
	s := authServer(t)
	enableAuth(t, s)
	req(t, s, "POST", "/api/auth/setup", `{"username":"admin","password":"correct-horse"}`, "")
	req(t, s, "POST", "/api/auth/logout", "", "")
	for i := 0; i < 5; i++ {
		req(t, s, "POST", "/api/auth/login", `{"username":"admin","password":"nope-nope"}`, "")
	}
	if w := req(t, s, "POST", "/api/auth/login", `{"username":"admin","password":"correct-horse"}`, ""); w.Code != 429 {
		t.Fatalf("6th login should be rate-limited, got %d", w.Code)
	}
}

func TestAuthSessionExpiryAndRotation(t *testing.T) {
	s := authServer(t)
	enableAuth(t, s)
	w := req(t, s, "POST", "/api/auth/setup", `{"username":"admin","password":"correct-horse"}`, "")
	cookie := sessionCookieOf(w)

	// Expired session -> 401.
	_, _ = s.db.Exec(`UPDATE sessions SET expires_at = ? WHERE token = ?`,
		time.Now().Add(-time.Hour).UTC().Format("2006-01-02T15:04:05.000Z"), cookie)
	if w := req(t, s, "GET", "/api/sections", "", cookie); w.Code != 401 {
		t.Fatalf("expired session should 401, got %d", w.Code)
	}

	// Near-expiry session rotates: response carries a new cookie.
	fresh := req(t, s, "POST", "/api/auth/login", `{"username":"admin","password":"correct-horse"}`, "")
	cookie = sessionCookieOf(fresh)
	_, _ = s.db.Exec(`UPDATE sessions SET expires_at = ? WHERE token = ?`,
		time.Now().Add(time.Hour).UTC().Format("2006-01-02T15:04:05.000Z"), cookie)
	w = req(t, s, "GET", "/api/sections", "", cookie)
	if w.Code != 200 {
		t.Fatalf("rotating request should pass, got %d", w.Code)
	}
	newCookie := sessionCookieOf(w)
	if newCookie == "" || newCookie == cookie {
		t.Fatal("expected session rotation to issue a new token")
	}
	// Old token is dead.
	if w := req(t, s, "GET", "/api/sections", "", cookie); w.Code != 401 {
		t.Fatalf("old token should be invalid after rotation, got %d", w.Code)
	}
}

func TestAuthDisable(t *testing.T) {
	s := authServer(t)
	enableAuth(t, s)
	w := req(t, s, "POST", "/api/auth/setup", `{"username":"admin","password":"correct-horse"}`, "")
	cookie := sessionCookieOf(w)

	// Unauthenticated disable is refused.
	if w := req(t, s, "POST", "/api/auth/disable", "", ""); w.Code != 401 {
		t.Fatalf("disable without session should 401, got %d", w.Code)
	}
	if w := req(t, s, "POST", "/api/auth/disable", "", cookie); w.Code != 204 {
		t.Fatalf("disable: %d", w.Code)
	}
	if w := req(t, s, "GET", "/api/sections", "", ""); w.Code != 200 {
		t.Fatalf("sections should reopen after disable, got %d", w.Code)
	}
}
