package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tdvorak/dash/internal/db"
	"go.uber.org/zap"
)

func hookServer(t *testing.T) *Server {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "dash.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	gin.SetMode(gin.TestMode)
	return newServer(zap.NewNop(), d, t.TempDir())
}

func hookReq(t *testing.T, s *Server, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", path, strings.NewReader(body))
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	s.routes().ServeHTTP(w, r)
	return w
}

func setHookSecret(t *testing.T, s *Server, body string) {
	t.Helper()
	w := req(t, s, "PUT", "/api/settings", body, "")
	if w.Code != 200 {
		t.Fatalf("set settings: %d %s", w.Code, w.Body.String())
	}
}

func TestGitHubHook(t *testing.T) {
	s := hookServer(t)
	payload := `{"repository":{"full_name":"o/r"}}`

	// No secret configured: endpoint must not exist.
	if w := hookReq(t, s, "/api/dev/hooks/github", payload, nil); w.Code != 404 {
		t.Fatalf("unconfigured hook should 404, got %d", w.Code)
	}

	setHookSecret(t, s, `{"dev.githubWebhookSecret":"s3cr3t"}`)

	// Bad signature rejected.
	if w := hookReq(t, s, "/api/dev/hooks/github", payload,
		map[string]string{"X-Hub-Signature-256": "sha256=deadbeef"}); w.Code != 401 {
		t.Fatalf("bad signature should 401, got %d", w.Code)
	}

	// Ping with valid signature.
	mac := hmac.New(sha256.New, []byte("s3cr3t"))
	mac.Write([]byte(`{"zen":"x"}`))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if w := hookReq(t, s, "/api/dev/hooks/github", `{"zen":"x"}`,
		map[string]string{
			"X-Hub-Signature-256": sig,
			"X-GitHub-Event":      "ping",
		}); w.Code != 200 {
		t.Fatalf("ping should 200, got %d", w.Code)
	}

	// Valid signature, unknown repo: accepted, no sync target.
	mac = hmac.New(sha256.New, []byte("s3cr3t"))
	mac.Write([]byte(payload))
	sig = "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if w := hookReq(t, s, "/api/dev/hooks/github", payload,
		map[string]string{"X-Hub-Signature-256": sig}); w.Code != 200 {
		t.Fatalf("valid hook should 200, got %d", w.Code)
	}
}

func TestGitLabHook(t *testing.T) {
	s := hookServer(t)
	payload := `{"project":{"path_with_namespace":"g/p"}}`

	if w := hookReq(t, s, "/api/dev/hooks/gitlab", payload, nil); w.Code != 404 {
		t.Fatalf("unconfigured hook should 404, got %d", w.Code)
	}

	setHookSecret(t, s, `{"dev.gitlabWebhookSecret":"tok"}`)

	if w := hookReq(t, s, "/api/dev/hooks/gitlab", payload,
		map[string]string{"X-Gitlab-Token": "wrong"}); w.Code != 401 {
		t.Fatalf("bad token should 401, got %d", w.Code)
	}

	if w := hookReq(t, s, "/api/dev/hooks/gitlab", payload,
		map[string]string{"X-Gitlab-Token": "tok"}); w.Code != 200 {
		t.Fatalf("valid hook should 200, got %d", w.Code)
	}
}
