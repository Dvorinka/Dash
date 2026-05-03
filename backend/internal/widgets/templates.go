package widgets

import (
	"bytes"
	"context"
	"dash/backend/internal/validation"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WidgetTemplate defines everything needed to support a widget type.
// To add a new widget:
//   1. Add a new entry to the All map below.
//   2. Implement Validate and Fetch functions in this file (or import them).
//   3. Add frontend template in frontend/lib/widgets/templates.ts.
//   4. Add frontend widget component in frontend/components/widgets/.
//   5. Add backend validation in validation.go WidgetType switch.
//   6. Update DB migration enum if needed.

type WidgetTemplate struct {
	Type           string
	Name           string
	Description    string
	Category       string // "system" | "service"
	DefaultTitle   string
	DefaultConfig  map[string]any
	NeedsDataFetch bool
	Validate       func(raw []byte) error
	Fetch          func(ctx context.Context, client *http.Client, raw []byte) ([]byte, error)
}

// All registered widget templates. Ordered slice for stable listing.
var All = []*WidgetTemplate{
	{
		Type:           "clock",
		Name:           "Clock",
		Description:    "Display current time across multiple timezones.",
		Category:       "system",
		DefaultTitle:   "Clock",
		DefaultConfig:  map[string]any{"timezones": []string{"UTC"}},
		NeedsDataFetch: false,
		Validate: func(raw []byte) error {
			var cfg struct {
				Timezones []string `json:"timezones"`
			}
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return err
			}
			if len(cfg.Timezones) == 0 {
				return errors.New("at least one timezone is required")
			}
			return nil
		},
	},
	{
		Type:           "image",
		Name:           "Image",
		Description:    "Show an image from a URL with an optional link.",
		Category:       "system",
		DefaultTitle:   "Image",
		DefaultConfig:  map[string]any{"imageUrl": "", "linkUrl": nil},
		NeedsDataFetch: false,
		Validate: func(raw []byte) error {
			var cfg struct {
				ImageURL string `json:"imageUrl"`
				LinkURL  string `json:"linkUrl"`
			}
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return err
			}
			if _, err := validation.AbsoluteHTTP(cfg.ImageURL, "imageUrl"); err != nil {
				return err
			}
			if cfg.LinkURL != "" {
				if _, err := validation.AbsoluteHTTP(cfg.LinkURL, "linkUrl"); err != nil {
					return err
				}
			}
			return nil
		},
	},
	{
		Type:           "pihole",
		Name:           "Pi-hole",
		Description:    "Live stats from a Pi-hole DNS sinkhole instance.",
		Category:       "service",
		DefaultTitle:   "Pi-hole",
		DefaultConfig:  map[string]any{"baseUrl": "", "apiToken": ""},
		NeedsDataFetch: true,
		Validate: func(raw []byte) error {
			var cfg struct {
				BaseURL  string `json:"baseUrl"`
				APIToken string `json:"apiToken"`
			}
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return err
			}
			if _, err := validation.AbsoluteHTTP(cfg.BaseURL, "baseUrl"); err != nil {
				return err
			}
			return nil
		},
		Fetch: fetchPiHole,
	},
	{
		Type:           "memos",
		Name:           "Memos",
		Description:    "Recent notes from your Memos instance.",
		Category:       "service",
		DefaultTitle:   "Memos",
		DefaultConfig:  map[string]any{"baseUrl": "", "apiToken": "", "pageSize": 5},
		NeedsDataFetch: true,
		Validate: func(raw []byte) error {
			var cfg struct {
				BaseURL  string `json:"baseUrl"`
				APIToken string `json:"apiToken"`
				PageSize int    `json:"pageSize"`
			}
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return err
			}
			if _, err := validation.AbsoluteHTTP(cfg.BaseURL, "baseUrl"); err != nil {
				return err
			}
			if cfg.APIToken == "" {
				return errors.New("apiToken is required")
			}
			return nil
		},
		Fetch: fetchMemos,
	},
	{
		Type:           "immich",
		Name:           "Immich",
		Description:    "Photo and video stats from your Immich server.",
		Category:       "service",
		DefaultTitle:   "Immich",
		DefaultConfig:  map[string]any{"baseUrl": "", "apiKey": ""},
		NeedsDataFetch: true,
		Validate: func(raw []byte) error {
			var cfg struct {
				BaseURL string `json:"baseUrl"`
				APIKey  string `json:"apiKey"`
			}
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return err
			}
			if _, err := validation.AbsoluteHTTP(cfg.BaseURL, "baseUrl"); err != nil {
				return err
			}
			if cfg.APIKey == "" {
				return errors.New("apiKey is required")
			}
			return nil
		},
		Fetch: fetchImmich,
	},
}

var byType = make(map[string]*WidgetTemplate)

func init() {
	for _, t := range All {
		byType[t.Type] = t
	}
}

func GetTemplate(widgetType string) (*WidgetTemplate, bool) {
	t, ok := byType[widgetType]
	return t, ok
}

func fetchPiHole(ctx context.Context, client *http.Client, raw []byte) ([]byte, error) {
	var cfg struct {
		BaseURL  string `json:"baseUrl"`
		APIToken string `json:"apiToken"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	base, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, errors.New("invalid Pi-hole baseUrl")
	}

	if payload, err := fetchPiHoleV6(ctx, client, base, cfg.APIToken); err == nil {
		return payload, nil
	}

	endpoints := []string{"/admin/api.php?summaryRaw"}
	var lastErr error
	var rawPayloadOut []byte
	for _, endpoint := range endpoints {
		requestURL := base.String() + endpoint
		if cfg.APIToken != "" {
			sep := "?"
			if strings.Contains(requestURL, "?") {
				sep = "&"
			}
			requestURL += sep + "auth=" + url.QueryEscape(cfg.APIToken)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, err
		}
		res, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		func() {
			defer res.Body.Close()
			if res.StatusCode < 200 || res.StatusCode >= 300 {
				lastErr = fmt.Errorf("Pi-hole returned %d", res.StatusCode)
				return
			}
			var rawPayload map[string]any
			if err := json.NewDecoder(res.Body).Decode(&rawPayload); err != nil {
				lastErr = err
				return
			}
			payload, err := normalizePiHole(rawPayload)
			if err != nil {
				lastErr = err
				return
			}
			lastErr = nil
			rawPayloadOut = payload
		}()
		if lastErr == nil && rawPayloadOut != nil {
			return rawPayloadOut, nil
		}
	}
	if lastErr == nil {
		lastErr = errors.New("Pi-hole fetch failed")
	}
	return nil, lastErr
}

func fetchPiHoleV6(ctx context.Context, client *http.Client, base *url.URL, password string) ([]byte, error) {
	sid := ""
	if password != "" {
		authURL := base.String() + "/api/auth"
		body, err := json.Marshal(map[string]string{"password": password})
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return nil, fmt.Errorf("Pi-hole auth returned %d", res.StatusCode)
		}
		var auth struct {
			Session struct {
				Valid bool   `json:"valid"`
				SID   string `json:"sid"`
			} `json:"session"`
		}
		if err := json.NewDecoder(res.Body).Decode(&auth); err != nil {
			return nil, err
		}
		if !auth.Session.Valid || auth.Session.SID == "" {
			return nil, errors.New("Pi-hole auth returned invalid session")
		}
		sid = auth.Session.SID
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String()+"/api/stats/summary", nil)
	if err != nil {
		return nil, err
	}
	if sid != "" {
		req.Header.Set("X-FTL-SID", sid)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("Pi-hole returned %d", res.StatusCode)
	}
	var rawPayload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&rawPayload); err != nil {
		return nil, err
	}
	return normalizePiHole(rawPayload)
}

func normalizePiHole(raw map[string]any) ([]byte, error) {
	blocked := number(raw, "queries_blocked")
	if blocked == 0 {
		blocked = nestedNumber(raw, "queries", "blocked")
	}
	if blocked == 0 {
		blocked = number(raw, "ads_blocked_today")
	}
	total := number(raw, "dns_queries_today")
	if total == 0 {
		total = nestedNumber(raw, "queries", "total")
	}
	if total == 0 {
		total = number(raw, "queries")
	}
	percent := number(raw, "ads_percentage_today")
	if percent == 0 {
		percent = nestedNumber(raw, "queries", "percent_blocked")
	}
	if percent == 0 && total > 0 {
		percent = blocked / total * 100
	}
	status := "unknown"
	if value, ok := raw["status"].(string); ok && value != "" {
		status = value
	}
	return json.Marshal(map[string]any{
		"blockedCount":   blocked,
		"queryCount":     total,
		"percentBlocked": percent,
		"status":         status,
		"fetchedAt":      time.Now().UTC().Format(time.RFC3339),
	})
}

func nestedNumber(raw map[string]any, objectKey string, key string) float64 {
	nested, ok := raw[objectKey].(map[string]any)
	if !ok {
		return 0
	}
	return number(nested, key)
}

func number(raw map[string]any, key string) float64 {
	switch value := raw[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case json.Number:
		out, _ := value.Float64()
		return out
	default:
		return 0
	}
}

func fetchMemos(ctx context.Context, client *http.Client, raw []byte) ([]byte, error) {
	var cfg struct {
		BaseURL  string `json:"baseUrl"`
		APIToken string `json:"apiToken"`
		PageSize int    `json:"pageSize"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	if cfg.PageSize <= 0 {
		cfg.PageSize = 5
	}
	if cfg.PageSize > 20 {
		cfg.PageSize = 20
	}

	reqURL := strings.TrimRight(cfg.BaseURL, "/") + fmt.Sprintf("/api/v1/memos?pageSize=%d", cfg.PageSize)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIToken)
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("memos returned %d", res.StatusCode)
	}

	var body struct {
		Memos []struct {
			Name       string `json:"name"`
			UID        string `json:"uid"`
			Content    string `json:"content"`
			CreateTime string `json:"createTime"`
			UpdateTime string `json:"updateTime"`
		} `json:"memos"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}

	type memoSummary struct {
		UID        string `json:"uid"`
		Content    string `json:"content"`
		CreateTime string `json:"createTime"`
	}
	summaries := make([]memoSummary, 0, len(body.Memos))
	for _, m := range body.Memos {
		content := strings.TrimSpace(m.Content)
		if len(content) > 120 {
			content = content[:117] + "..."
		}
		summaries = append(summaries, memoSummary{
			UID:        m.UID,
			Content:    content,
			CreateTime: m.CreateTime,
		})
	}

	return json.Marshal(map[string]any{
		"memos":     summaries,
		"count":     len(body.Memos),
		"fetchedAt": time.Now().UTC().Format(time.RFC3339),
	})
}

func fetchImmich(ctx context.Context, client *http.Client, raw []byte) ([]byte, error) {
	var cfg struct {
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}

	reqURL := strings.TrimRight(cfg.BaseURL, "/") + "/api/server-info/statistics"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("immich returned %d", res.StatusCode)
	}

	var body struct {
		Photos int `json:"photos"`
		Videos int `json:"videos"`
		Usage  int `json:"usage"` // bytes
		Users  int `json:"users"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}

	// Format usage to human readable
	usageStr := formatBytes(body.Usage)

	return json.Marshal(map[string]any{
		"photos":    body.Photos,
		"videos":    body.Videos,
		"usage":     usageStr,
		"usageRaw":  body.Usage,
		"users":     body.Users,
		"fetchedAt": time.Now().UTC().Format(time.RFC3339),
	})
}

func formatBytes(b int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
