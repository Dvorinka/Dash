package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Provider webhooks — GitHub (HMAC signature) and GitLab (token header).
// They live under /api/dev/hooks/ which publicAPI exempts from session
// auth; the webhook secret is the credential instead. With no secret
// configured the endpoints 404 so they can't be probed.

func (s *Server) devHookGitHub(c *gin.Context) {
	secret := s.settingStr("dev.githubWebhookSecret")
	if secret == "" {
		c.Status(http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 4<<20))
	if err != nil {
		fail(c, http.StatusBadRequest, "bad body")
		return
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Hub-Signature-256")), []byte(want)) != 1 {
		fail(c, http.StatusUnauthorized, "bad signature")
		return
	}
	if c.GetHeader("X-GitHub-Event") == "ping" {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	var payload struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if json.Unmarshal(body, &payload) != nil || payload.Repository.FullName == "" {
		c.Status(http.StatusAccepted)
		return
	}
	s.devHookSync("github", payload.Repository.FullName)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) devHookGitLab(c *gin.Context) {
	secret := s.settingStr("dev.gitlabWebhookSecret")
	if secret == "" {
		c.Status(http.StatusNotFound)
		return
	}
	if subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Gitlab-Token")), []byte(secret)) != 1 {
		fail(c, http.StatusUnauthorized, "bad token")
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 4<<20))
	if err != nil {
		fail(c, http.StatusBadRequest, "bad body")
		return
	}
	var payload struct {
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
	}
	if json.Unmarshal(body, &payload) != nil || payload.Project.PathWithNamespace == "" {
		c.Status(http.StatusAccepted)
		return
	}
	s.devHookSync("gitlab", payload.Project.PathWithNamespace)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// devHookSync kicks a sync for every enabled project matching the repo.
func (s *Server) devHookSync(provider, path string) {
	rows, err := s.db.Query(
		`SELECT id FROM dev_projects WHERE provider=? AND path=? AND enabled=1`, provider, path)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			go s.syncDevProject(context.Background(), id)
		}
	}
}
