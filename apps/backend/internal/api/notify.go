package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Notifications: a small transport registry. Each transport is configured via
// the settings key notify_<name> holding a JSON object of fields; a transport
// is active when its required fields are set. Multiple transports may be
// active — notify() fans out to all of them.
//
// notify_webhook keeps back-compat: a plain string value means {"url": value}.
//
// Secrets (smtp_pass, tokens) live in settings like any other value and are
// never logged. Single-user self-hosted is the threat model; P5d auth gates
// the settings endpoint when enabled.

type notifyPayload struct {
	Event   string `json:"event"`
	Name    string `json:"name"`
	Text    string `json:"text"`    // Slack-style
	Content string `json:"content"` // Discord-style
	Message string `json:"message"` // ntfy/generic
	At      string `json:"at"`
}

type transport struct {
	name     string
	required []string
	send     func(ctx context.Context, cfg map[string]string, p notifyPayload) error
}

var transports = []transport{
	{"webhook", []string{"url"}, sendWebhook},
	{"slack", []string{"url"}, sendSlack},
	{"discord", []string{"url"}, sendDiscord},
	{"telegram", []string{"token", "chat_id"}, sendTelegram},
	{"gotify", []string{"server", "token"}, sendGotify},
	{"ntfy", []string{"server", "topic"}, sendNtfy},
	{"smtp", []string{"host", "port", "from", "to"}, sendSMTP},
}

// notifyCfg loads notify_<name> from settings. Values are JSON: an object for
// presets, or a bare string (legacy webhook) meaning {"url": s}.
func (s *Server) notifyCfg(name string) map[string]string {
	var raw string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key = 'notify_`+name+`'`).Scan(&raw); err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		var str string
		if err := json.Unmarshal([]byte(raw), &str); err == nil && name == "webhook" {
			return map[string]string{"url": str}
		}
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if str, ok := v.(string); ok {
			out[k] = str
		}
	}
	return out
}

// configured reports whether all required fields are present. Partially
// configured transports count as configured so their send error surfaces.
func (t transport) configured(cfg map[string]string) bool {
	if cfg == nil {
		return false
	}
	for _, k := range t.required {
		if strings.TrimSpace(cfg[k]) == "" {
			return false
		}
	}
	return true
}

// notify fans an event out to every configured transport, asynchronously.
func (s *Server) notify(event, name, msg string) {
	p := notifyPayload{Event: event, Name: name, Text: msg, Content: msg, Message: msg,
		At: time.Now().UTC().Format(time.RFC3339)}
	for _, tr := range transports {
		cfg := s.notifyCfg(tr.name)
		if !tr.configured(cfg) {
			continue
		}
		tr, cfg := tr, cfg
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := tr.send(ctx, cfg, p); err != nil {
				s.log.Warn("notify failed", zap.String("transport", tr.name),
					zap.String("event", event), zap.Error(err))
			}
		}()
	}
}

// postJSON posts a JSON body and reports non-2xx as an error.
func postJSON(ctx context.Context, url string, payload any, headers map[string]string) error {
	if parseHTTPURL(url) == "" {
		return fmt.Errorf("not an http(s) URL")
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("returned %d", resp.StatusCode)
	}
	return nil
}

func sendWebhook(ctx context.Context, cfg map[string]string, p notifyPayload) error {
	return postJSON(ctx, cfg["url"], p, map[string]string{
		"Title": p.Text, "X-Dash-Event": p.Event,
	})
}

func sendSlack(ctx context.Context, cfg map[string]string, p notifyPayload) error {
	return postJSON(ctx, cfg["url"], map[string]any{"text": p.Text}, nil)
}

func sendDiscord(ctx context.Context, cfg map[string]string, p notifyPayload) error {
	return postJSON(ctx, cfg["url"], map[string]any{"content": p.Content}, nil)
}

func sendTelegram(ctx context.Context, cfg map[string]string, p notifyPayload) error {
	base := strings.TrimRight(cfg["server"], "/") // optional self-hosted bot-api
	if base == "" {
		base = "https://api.telegram.org"
	}
	url := base + "/bot" + cfg["token"] + "/sendMessage"
	return postJSON(ctx, url, map[string]any{
		"chat_id": cfg["chat_id"], "text": p.Text,
	}, nil)
}

func sendGotify(ctx context.Context, cfg map[string]string, p notifyPayload) error {
	server := strings.TrimRight(cfg["server"], "/")
	url := server + "/message?token=" + cfg["token"]
	return postJSON(ctx, url, map[string]any{
		"title": p.Name, "message": p.Message, "priority": 5,
	}, nil)
}

func sendNtfy(ctx context.Context, cfg map[string]string, p notifyPayload) error {
	server := strings.TrimRight(cfg["server"], "/")
	url := server + "/" + strings.TrimLeft(cfg["topic"], "/")
	if parseHTTPURL(url) == "" {
		return fmt.Errorf("not an http(s) URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(p.Message))
	if err != nil {
		return err
	}
	req.Header.Set("Title", p.Name)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("returned %d", resp.StatusCode)
	}
	return nil
}

// sendSMTP uses net/smtp with a hard deadline, opportunistic STARTTLS, and
// plain auth only when a user is configured.
func sendSMTP(ctx context.Context, cfg map[string]string, p notifyPayload) error {
	host, port := cfg["host"], cfg["port"]
	addr := net.JoinHostPort(host, port)
	d := net.Dialer{Timeout: 10 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	} else {
		_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}
	if cfg["user"] != "" {
		if err := c.Auth(smtp.PlainAuth("", cfg["user"], cfg["pass"], host)); err != nil {
			return err
		}
	}
	if err := c.Mail(cfg["from"]); err != nil {
		return err
	}
	for _, rcpt := range strings.Split(cfg["to"], ",") {
		if err := c.Rcpt(strings.TrimSpace(rcpt)); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: [Dash] %s\r\n\r\n%s\r\n",
		cfg["from"], cfg["to"], p.Text, p.Message)
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// parseHTTPURL enforces http/https on the configured webhook — same SSRF
// posture as monitor targets (user-configured, but no file:// or friends).
func parseHTTPURL(raw string) string {
	if !validURL(raw) {
		return ""
	}
	return raw
}

// notifyTest sends a test event synchronously so the UI gets a real result.
// Optional body {"transport": "smtp"} limits the test to one transport.
func (s *Server) notifyTest(c *gin.Context) {
	var in struct {
		Transport string `json:"transport"`
	}
	_ = c.ShouldBindJSON(&in)
	p := notifyPayload{Event: "test", Name: "dash", Text: "Dash test notification",
		Content: "Dash test notification", Message: "Dash test notification",
		At: time.Now().UTC().Format(time.RFC3339)}

	results := map[string]string{}
	for _, tr := range transports {
		if in.Transport != "" && tr.name != in.Transport {
			continue
		}
		cfg := s.notifyCfg(tr.name)
		if !tr.configured(cfg) {
			if in.Transport != "" {
				fail(c, http.StatusBadRequest, tr.name+" not configured")
				return
			}
			continue
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		err := tr.send(ctx, cfg, p)
		cancel()
		if err != nil {
			results[tr.name] = "error: " + err.Error()
		} else {
			results[tr.name] = "ok"
		}
	}
	if len(results) == 0 {
		fail(c, http.StatusBadRequest, "no notification transport configured")
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}
