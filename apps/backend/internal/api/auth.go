package api

// Opt-in local auth — disabled by default; enabling creates the first admin.
// Local username + bcrypt password only (no OIDC/SSO — deliberate non-goal).
// Sessions are random tokens in SQLite behind an httpOnly SameSite=Lax cookie;
// login is rate-limited per IP. Public endpoints (status pages, badges, push,
// agent ingest, metrics) bypass the middleware.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookie = "dash_session"
	sessionTTL    = 7 * 24 * time.Hour
	// Rotate when less than half the TTL remains.
	sessionRotate = sessionTTL / 2
)

func (s *Server) authEnabled() bool {
	var raw string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key = 'auth_enabled'`).Scan(&raw); err != nil {
		return false
	}
	var on bool
	_ = json.Unmarshal([]byte(raw), &on)
	return on
}

func (s *Server) userCount() int {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n
}

// ---------- sessions ----------

func (s *Server) createSession(c *gin.Context, userID string) {
	var tok [32]byte
	_, _ = rand.Read(tok[:])
	token := hex.EncodeToString(tok[:])
	exp := time.Now().Add(sessionTTL)
	_, _ = s.db.Exec(`INSERT INTO sessions (token, user_id, expires_at) VALUES (?,?,?)`,
		token, userID, exp.UTC().Format("2006-01-02T15:04:05.000Z"))
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,
		Secure:   c.Request.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

// sessionUser resolves the cookie to a user, sliding-rotating near expiry.
// Returns "" when absent/expired.
func (s *Server) sessionUser(c *gin.Context) string {
	ck, err := c.Cookie(sessionCookie)
	if err != nil || ck == "" {
		return ""
	}
	var userID, expires string
	err = s.db.QueryRow(`SELECT user_id, expires_at FROM sessions WHERE token = ?`, ck).Scan(&userID, &expires)
	if err != nil {
		return ""
	}
	exp, err := time.Parse("2006-01-02T15:04:05.000Z", expires)
	if err != nil || time.Now().After(exp) {
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE token = ?`, ck)
		return ""
	}
	if time.Until(exp) < sessionRotate {
		// Rotate: drop the old token, issue a fresh one.
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE token = ?`, ck)
		s.createSession(c, userID)
	}
	return userID
}

// ---------- rate limiting (login only, in-memory per-IP) ----------

var loginMu sync.Mutex
var loginHits = map[string][]time.Time{}

func loginAllowed(ip string) bool {
	loginMu.Lock()
	defer loginMu.Unlock()
	cut := time.Now().Add(-time.Minute)
	kept := loginHits[ip][:0]
	for _, t := range loginHits[ip] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= 5 {
		loginHits[ip] = kept
		return false
	}
	loginHits[ip] = append(kept, time.Now())
	return true
}

// ---------- middleware ----------

// publicAPI lists /api paths reachable without a session. Auth endpoints must
// stay open (login itself); push/ingest/badge/metrics/public status carry
// their own tokens or are intentionally public.
func publicAPI(path string) bool {
	switch {
	case strings.HasPrefix(path, "/api/auth/"),
		strings.HasPrefix(path, "/api/push/"),
		path == "/api/systems/ingest",
		strings.HasPrefix(path, "/api/badge/"),
		path == "/api/metrics",
		strings.HasPrefix(path, "/api/healthz"),
		strings.HasSuffix(path, "/public"):
		return true
	}
	return false
}

// requireAuth gates every /api route when the auth_enabled setting is on.
func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.authEnabled() || publicAPI(c.Request.URL.Path) {
			c.Next()
			return
		}
		if s.sessionUser(c) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		c.Next()
	}
}

// ---------- handlers ----------

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// sessionInfo reports auth state for the frontend shell — always public so
// the UI can decide between the board and the login screen.
func (s *Server) authSession(c *gin.Context) {
	on := s.authEnabled()
	out := gin.H{"enabled": on, "authed": false, "setup": false}
	if !on {
		c.JSON(http.StatusOK, out)
		return
	}
	out["setup"] = s.userCount() == 0
	if uid := s.sessionUser(c); uid != "" {
		var name string
		_ = s.db.QueryRow(`SELECT username FROM users WHERE id = ?`, uid).Scan(&name)
		out["authed"] = true
		out["username"] = name
	}
	c.JSON(http.StatusOK, out)
}

// setup creates the first user — only while auth is on and no users exist.
func (s *Server) authSetup(c *gin.Context) {
	if !s.authEnabled() {
		fail(c, http.StatusBadRequest, "auth is not enabled")
		return
	}
	if s.userCount() > 0 {
		fail(c, http.StatusConflict, "admin user already exists")
		return
	}
	var in credentials
	if err := c.ShouldBindJSON(&in); err != nil ||
		len(strings.TrimSpace(in.Username)) < 2 || len(in.Password) < 8 {
		fail(c, http.StatusBadRequest, "username (2+) and password (8+) required")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, "hash failed")
		return
	}
	uid := newID("u")
	if _, err := s.db.Exec(`INSERT INTO users (id, username, pass_hash) VALUES (?,?,?)`,
		uid, strings.TrimSpace(in.Username), string(hash)); err != nil {
		fail(c, http.StatusConflict, "username taken")
		return
	}
	s.createSession(c, uid)
	c.JSON(http.StatusCreated, gin.H{"username": in.Username})
}

func (s *Server) authLogin(c *gin.Context) {
	if !s.authEnabled() {
		fail(c, http.StatusBadRequest, "auth is not enabled")
		return
	}
	ip := c.ClientIP()
	if !loginAllowed(ip) {
		fail(c, http.StatusTooManyRequests, "too many attempts — wait a minute")
		return
	}
	var in credentials
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	var uid, hash string
	err := s.db.QueryRow(`SELECT id, pass_hash FROM users WHERE username = ?`,
		strings.TrimSpace(in.Username)).Scan(&uid, &hash)
	if err != nil {
		// Unknown user — run bcrypt anyway so timing doesn't leak existence.
		hash = "$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5sA5t6p7K0dGm7ZC4yQ0yCk0cFk0K"
	}
	// CompareHashAndPassword is constant-time; err != nil covers both
	// unknown user (dummy hash above) and wrong password.
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		fail(c, http.StatusUnauthorized, "invalid credentials")
		return
	}
	s.createSession(c, uid)
	c.JSON(http.StatusOK, gin.H{"username": in.Username})
}

func (s *Server) authLogout(c *gin.Context) {
	if ck, err := c.Cookie(sessionCookie); err == nil {
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE token = ?`, ck)
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
	})
	c.Status(http.StatusNoContent)
}

// authDisable turns auth off and wipes users+sessions. Requires a session —
// you can't switch the lock off from outside.
func (s *Server) authDisable(c *gin.Context) {
	if s.sessionUser(c) == "" {
		fail(c, http.StatusUnauthorized, "authentication required")
		return
	}
	_, _ = s.db.Exec(`DELETE FROM sessions`)
	_, _ = s.db.Exec(`DELETE FROM users`)
	if _, err := s.db.Exec(`INSERT INTO settings (key, value) VALUES ('auth_enabled','false')
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`); err != nil {
		s.log.Error("auth disable", zap.Error(err))
	}
	c.Status(http.StatusNoContent)
}
