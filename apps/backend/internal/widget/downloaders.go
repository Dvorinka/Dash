package widget

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Download clients and seed boxes — SABnzbd, NZBGet, qui — each with its own
// auth scheme. Patterns follow dashbrr's sabnzbd/nzbget/qui services.

// flexString accepts a JSON string or number and yields the string form —
// SABnzbd changed queue counters from strings to numbers upstream.
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	b = []byte(s)
	if len(b) == 0 || string(b) == "null" {
		*f = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal([]byte(b), &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal([]byte(b), &n); err != nil {
		return err
	}
	*f = flexString(n)
	return nil
}

func flexInt(f flexString) int {
	n, _ := strconv.Atoi(strings.TrimSpace(string(f)))
	return n
}

// sabnzbdWidget: /api?mode=...&output=json&apikey=<key>.
type sabnzbdWidget struct{}

func init() { Register(sabnzbdWidget{}) }

func (sabnzbdWidget) Meta() Type {
	return Type{
		Type:        "sabnzbd",
		Name:        "SABnzbd",
		Description: "Queue, speed and version",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://sabnzbd.local:8080", Required: true},
			{Key: "apiKey", Label: "API key", Secret: true, Required: true},
		},
	}
}

func (sabnzbdWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	key := cfgStr(cfg, "apiKey")
	if endpoint == "" || key == "" {
		return nil, fmt.Errorf("endpoint and apiKey required")
	}
	api := func(mode string) string {
		return endpoint + "/api?mode=" + mode + "&output=json&apikey=" + url.QueryEscape(key)
	}

	var ver struct {
		Version string `json:"version"`
	}
	if err := getJSON(ctx, api("version"), nil, &ver); err != nil {
		return nil, err
	}

	d := &svcData{Status: "online", Version: ver.Version}

	var q struct {
		Queue struct {
			NoOfSlots flexString `json:"noofslots"`
			Speed     string     `json:"speed"`
			Paused    bool       `json:"paused"`
		} `json:"queue"`
	}
	if getJSON(ctx, api("queue"), nil, &q) == nil {
		d.QueueTotal = flexInt(q.Queue.NoOfSlots)
		d.Speed = q.Queue.Speed
		d.Paused = q.Queue.Paused
		if d.Paused {
			d.Status = "warning"
			d.Warning = "queue paused"
			d.Issues = 1
		}
	}
	return d, nil
}

// nzbgetWidget: JSON-RPC POST to /jsonrpc with HTTP Basic auth. apiKey may be
// "user:pass", a bare password (user defaults to "nzbget"), or credentials may
// ride in the endpoint URL — same rules as dashbrr.
type nzbgetWidget struct{}

func init() { Register(nzbgetWidget{}) }

func (nzbgetWidget) Meta() Type {
	return Type{
		Type:        "nzbget",
		Name:        "NZBGet",
		Description: "Download rate and queue state",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://nzbget.local:6789", Required: true},
			{Key: "apiKey", Label: "Password", Secret: true, Placeholder: "password or user:pass", Required: true},
		},
	}
}

func (nzbgetWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	raw := strings.TrimSpace(cfgStr(cfg, "endpoint"))
	key := strings.TrimSpace(cfgStr(cfg, "apiKey"))
	if raw == "" || key == "" {
		return nil, fmt.Errorf("endpoint and apiKey required")
	}

	base, user, pass, err := nzbgetCreds(raw, key)
	if err != nil {
		return nil, err
	}
	hdr := map[string]string{
		"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass)),
	}
	rpc := func(method string, out any) error {
		return postJSON(ctx, strings.TrimRight(base, "/")+"/jsonrpc", hdr,
			map[string]any{"method": method}, out)
	}

	var statusResp struct {
		Result struct {
			DownloadPaused bool `json:"DownloadPaused"`
			PostPaused     bool `json:"PostPaused"`
			ScanPaused     bool `json:"ScanPaused"`
			QuotaReached   bool `json:"QuotaReached"`
			DownloadRate   int64 `json:"DownloadRate"`
		} `json:"result"`
	}
	if err := rpc("status", &statusResp); err != nil {
		return nil, err
	}

	d := &svcData{Status: "online"}
	st := statusResp.Result
	for _, p := range []struct {
		on   bool
		msg  string
	}{{st.DownloadPaused, "download paused"}, {st.PostPaused, "post-processing paused"}, {st.ScanPaused, "scan paused"}, {st.QuotaReached, "quota reached"}} {
		if p.on {
			d.Issues++
			if d.Warning == "" {
				d.Warning = p.msg
			}
		}
	}
	if d.Issues > 0 {
		d.Status = "warning"
	}
	if st.DownloadRate > 0 {
		d.Speed = formatBytesRate(st.DownloadRate)
	}

	// Version is best-effort; the result is a bare JSON string.
	var verResp struct {
		Result json.RawMessage `json:"result"`
	}
	if rpc("version", &verResp) == nil {
		var v string
		if json.Unmarshal(verResp.Result, &v) == nil {
			d.Version = v
		}
	}
	return d, nil
}

func nzbgetCreds(raw, key string) (base, user, pass string, err error) {
	u, perr := url.Parse(raw)
	if perr != nil || u.Scheme == "" || u.Host == "" {
		return "", "", "", fmt.Errorf("invalid endpoint URL")
	}
	user, pass = "", ""
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
		u.User = nil
	}
	if user == "" || pass == "" {
		switch {
		case strings.Contains(key, ":"):
			parts := strings.SplitN(key, ":", 2)
			user, pass = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		default:
			user, pass = "nzbget", key
		}
	}
	if user == "" || pass == "" {
		return "", "", "", fmt.Errorf("credentials required")
	}
	return u.String(), user, pass, nil
}

func formatBytesRate(bps int64) string {
	switch {
	case bps >= 1<<30:
		return fmt.Sprintf("%.1f GB/s", float64(bps)/(1<<30))
	case bps >= 1<<20:
		return fmt.Sprintf("%.1f MB/s", float64(bps)/(1<<20))
	case bps >= 1<<10:
		return fmt.Sprintf("%.0f KB/s", float64(bps)/(1<<10))
	default:
		return fmt.Sprintf("%d B/s", bps)
	}
}

// quiWidget: X-API-Key, /health probe, /api/instances for connected counts.
type quiWidget struct{}

func init() { Register(quiWidget{}) }

func (quiWidget) Meta() Type {
	return Type{
		Type:        "qui",
		Name:        "qui",
		Description: "Cross-seed instances online",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://qui.local:7476", Required: true},
			{Key: "apiKey", Label: "API key", Secret: true, Required: true},
		},
	}
}

func (quiWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	key := cfgStr(cfg, "apiKey")
	if endpoint == "" || key == "" {
		return nil, fmt.Errorf("endpoint and apiKey required")
	}
	hdr := map[string]string{"X-API-Key": key}

	// /health is the liveness probe.
	var health json.RawMessage
	if err := getJSON(ctx, endpoint+"/health", hdr, &health); err != nil {
		return nil, err
	}

	d := &svcData{Status: "online"}
	var instances []struct {
		Connected bool `json:"connected"`
		IsActive  bool `json:"isActive"`
	}
	if getJSON(ctx, endpoint+"/api/instances", hdr, &instances) == nil {
		d.Total = len(instances)
		for _, in := range instances {
			if in.Connected {
				d.Up++
			}
		}
		if d.Total > 0 && d.Up < d.Total {
			d.Status = "warning"
			d.Issues = d.Total - d.Up
			d.Warning = fmt.Sprintf("%d instance(s) disconnected", d.Issues)
		}
	}
	return d, nil
}
