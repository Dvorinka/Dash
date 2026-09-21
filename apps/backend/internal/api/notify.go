package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Notifications: one outbound webhook, configured via the notify_webhook
// settings key. Payload carries event/name/text/content/message so the same
// POST satisfies Slack ({text}), Discord ({content}), ntfy, and generic
// receivers. SMTP and per-service channels are deliberately out of scope —
// a webhook bridge (e.g. ntfy, Gotify, shoutrrr) covers them.

type notifyPayload struct {
	Event   string `json:"event"`
	Name    string `json:"name"`
	Text    string `json:"text"`    // Slack-style
	Content string `json:"content"` // Discord-style
	Message string `json:"message"` // ntfy/generic
	At      string `json:"at"`
}

func (s *Server) webhookURL() string {
	var raw string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key = 'notify_webhook'`).Scan(&raw); err != nil {
		return ""
	}
	// Settings values are stored as JSON — unwrap the quoted string.
	var u string
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		return ""
	}
	return u
}

// notify fires the webhook asynchronously; delivery failures are logged, never fatal.
func (s *Server) notify(event, name, msg string) {
	url := s.webhookURL()
	if url == "" {
		return
	}
	if u := parseHTTPURL(url); u == "" {
		s.log.Warn("notify_webhook is not an http(s) URL", zap.String("event", event))
		return
	}
	p := notifyPayload{Event: event, Name: name, Text: msg, Content: msg, Message: msg,
		At: time.Now().UTC().Format(time.RFC3339)}
	go s.deliver(url, p)
}

func (s *Server) deliver(url string, p notifyPayload) {
	body, _ := json.Marshal(p)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Title", p.Text)  // ntfy reads the title header
	req.Header.Set("X-Dash-Event", p.Event)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.log.Warn("webhook delivery failed", zap.String("event", p.Event), zap.Error(err))
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		s.log.Warn("webhook rejected", zap.String("event", p.Event), zap.Int("status", resp.StatusCode))
	}
}

// parseHTTPURL enforces http/https on the configured webhook — same SSRF
// posture as monitor targets (user-configured, but no file:// or friends).
func parseHTTPURL(raw string) string {
	if !validURL(raw) {
		return ""
	}
	return raw
}

// notifyTest lets the settings UI verify the configured webhook.
func (s *Server) notifyTest(c *gin.Context) {
	if s.webhookURL() == "" {
		fail(c, http.StatusBadRequest, "notify_webhook not configured")
		return
	}
	s.notify("test", "dash", "Dash test notification")
	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}
