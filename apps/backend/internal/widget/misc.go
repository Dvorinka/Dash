package widget

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// Misc services — Uptime Kuma, Traefik, Maintainerr. Patterns follow
// dashbrr's uptimekuma/traefik/maintainerr services.

// kumaWidget reads Uptime Kuma's Prometheus /metrics endpoint. Auth is HTTP
// Basic — dashbrr sends the API key as the password; Kuma accepts any username.
type kumaWidget struct{}

func init() { Register(kumaWidget{}) }

func (kumaWidget) Meta() Type {
	return Type{
		Type:        "uptimekuma",
		Name:        "Uptime Kuma",
		Description: "Monitor up/down counts from /metrics",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://kuma.local:3001", Required: true},
			{Key: "apiKey", Label: "Metrics password", Secret: true, Required: true},
		},
	}
}

func (kumaWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	key := cfgStr(cfg, "apiKey")
	if endpoint == "" || key == "" {
		return nil, fmt.Errorf("endpoint and apiKey required")
	}
	hdr := map[string]string{
		"Accept":        "text/plain",
		"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("dash:"+key)),
	}
	code, body, err := getRaw(ctx, endpoint+"/metrics", hdr)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("upstream returned %d", code)
	}

	d := &svcData{Status: "online"}
	var down []string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "monitor_status{") {
			continue
		}
		d.Total++
		if strings.HasSuffix(line, " 1") {
			d.Up++
			continue
		}
		if name := kumaLabel(line, "monitor_name"); name != "" {
			down = append(down, name)
		} else {
			down = append(down, "monitor")
		}
	}
	if len(down) > 0 {
		d.Status = "warning"
		d.Issues = len(down)
		d.Warning = strings.Join(down[:min(3, len(down))], ", ")
		if len(down) > 3 {
			d.Warning += fmt.Sprintf(" +%d more", len(down)-3)
		}
	}
	return d, nil
}

// kumaLabel extracts a {key="value"} label from a prometheus metric line.
func kumaLabel(line, key string) string {
	needle := key + `="`
	i := strings.Index(line, needle)
	if i < 0 {
		return ""
	}
	rest := line[i+len(needle):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// traefikWidget: /api/version + /api/overview. apiKey is optional — Traefik's
// API is unauthenticated unless secured; the key becomes Authorization
// ("Bearer x", "user:pass" → Basic, raw → Bearer), same as dashbrr.
type traefikWidget struct{}

func init() { Register(traefikWidget{}) }

func (traefikWidget) Meta() Type {
	return Type{
		Type:        "traefik",
		Name:        "Traefik",
		Description: "Version and router health",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://traefik.local:8080", Required: true},
			{Key: "apiKey", Label: "API key / token", Secret: true, Placeholder: "optional"},
		},
	}
}

func traefikAuth(key string) string {
	token := strings.TrimSpace(key)
	if token == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return token
	}
	if strings.Contains(token, ":") {
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(token))
	}
	return "Bearer " + token
}

func (traefikWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint required")
	}
	var hdr map[string]string
	if auth := traefikAuth(cfgStr(cfg, "apiKey")); auth != "" {
		hdr = map[string]string{"Authorization": auth}
	}

	var ver struct {
		Version string `json:"version"`
	}
	if err := getJSON(ctx, endpoint+"/api/version", hdr, &ver); err != nil {
		return nil, err
	}

	d := &svcData{Status: "online", Version: ver.Version}

	var ov struct {
		HTTP struct {
			Routers struct {
				Total    int `json:"total"`
				Warnings int `json:"warnings"`
				Errors   int `json:"errors"`
			} `json:"routers"`
		} `json:"http"`
	}
	if getJSON(ctx, endpoint+"/api/overview", hdr, &ov) == nil {
		d.Total = ov.HTTP.Routers.Total
		d.Issues = ov.HTTP.Routers.Warnings + ov.HTTP.Routers.Errors
		if d.Issues > 0 {
			d.Status = "warning"
			d.Warning = fmt.Sprintf("%d router warning(s)/error(s)", d.Issues)
		}
	}
	return d, nil
}

// maintainerrWidget: /api/app/status is unauthenticated and returns version +
// updateAvailable. Collections need an API key upstream but add nothing a
// status tile needs — skipped per ponytail.
type maintainerrWidget struct{}

func init() { Register(maintainerrWidget{}) }

func (maintainerrWidget) Meta() Type {
	return Type{
		Type:        "maintainerr",
		Name:        "Maintainerr",
		Description: "Version and update status",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://maintainerr.local:6246", Required: true},
		},
	}
}

func (maintainerrWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint required")
	}
	var st struct {
		Version         string `json:"version"`
		UpdateAvailable bool   `json:"updateAvailable"`
	}
	if err := getJSON(ctx, endpoint+"/api/app/status", nil, &st); err != nil {
		return nil, err
	}
	return &svcData{
		Status:          "online",
		Version:         st.Version,
		UpdateAvailable: st.UpdateAvailable,
		Issues:          0,
	}, nil
}
