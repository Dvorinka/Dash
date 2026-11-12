package widget

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Homelab connections that share the svcData tile: a torrent client, a
// hypervisor, a container panel, a proxy manager, a home controller, and a
// sync daemon. Each one is a couple of authenticated GETs.

func init() {
	Register(qbittorrentWidget{})
	Register(proxmoxWidget{})
	Register(portainerWidget{})
	Register(npmWidget{})
	Register(homeAssistantWidget{})
	Register(syncthingWidget{})
}

func call(ctx context.Context, method, rawURL string, headers map[string]string, body io.Reader, contentType string) (int, http.Header, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return 0, nil, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("upstream unreachable")
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return resp.StatusCode, resp.Header, nil, err
	}
	return resp.StatusCode, resp.Header, b, nil
}

func okJSON(code int, body []byte, out any) error {
	if code != http.StatusOK {
		return fmt.Errorf("upstream returned %d", code)
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func endpointOf(cfg json.RawMessage) (string, error) {
	e := strings.TrimRight(cfgStr(cfg, "endpoint"), "/")
	if e == "" {
		return "", fmt.Errorf("endpoint required")
	}
	return e, nil
}

func rate(bps float64) string {
	switch {
	case bps >= 1024*1024:
		return fmt.Sprintf("%.1f MB/s", bps/1024/1024)
	case bps >= 1024:
		return fmt.Sprintf("%.0f KB/s", bps/1024)
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}

func cookieHeader(h http.Header) string {
	var parts []string
	for _, c := range h.Values("Set-Cookie") {
		parts = append(parts, strings.Split(c, ";")[0])
	}
	return strings.Join(parts, "; ")
}

// qbittorrentWidget logs in with the WebUI form, then reads transfer speed
// and torrent states. apiKey is the WebUI password.
type qbittorrentWidget struct{}

func (qbittorrentWidget) Meta() Type {
	return Type{
		Type:        "qbittorrent",
		Name:        "qBittorrent",
		Description: "Active torrents and transfer speed",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://qbittorrent.local:8080", Required: true},
			{Key: "username", Label: "Username", Placeholder: "admin", Required: true},
			{Key: "apiKey", Label: "Password", Secret: true, Required: true},
		},
	}
}

func (qbittorrentWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint, err := endpointOf(cfg)
	if err != nil {
		return nil, err
	}
	user, pass := cfgStr(cfg, "username"), cfgStr(cfg, "apiKey")
	if user == "" || pass == "" {
		return nil, fmt.Errorf("username and password required")
	}
	form := url.Values{"username": {user}, "password": {pass}}
	code, hdr, body, err := call(ctx, http.MethodPost, endpoint+"/api/v2/auth/login", nil, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK || !strings.Contains(string(body), "Ok") {
		return nil, fmt.Errorf("upstream returned %d", code)
	}
	auth := map[string]string{"Cookie": cookieHeader(hdr)}

	code, _, body, err = call(ctx, http.MethodGet, endpoint+"/api/v2/transfer/info", auth, nil, "")
	if err != nil {
		return nil, err
	}
	var xfer struct {
		Dl float64 `json:"dl_info_speed"`
		Ul float64 `json:"ul_info_speed"`
	}
	if err := okJSON(code, body, &xfer); err != nil {
		return nil, err
	}
	code, _, body, err = call(ctx, http.MethodGet, endpoint+"/api/v2/torrents/info", auth, nil, "")
	if err != nil {
		return nil, err
	}
	var tors []struct {
		Name  string `json:"name"`
		State string `json:"state"`
	}
	if err := okJSON(code, body, &tors); err != nil {
		return nil, err
	}

	d := &svcData{Status: "online"}
	var bad []string
	active := 0
	paused := len(tors) > 0
	for _, tor := range tors {
		switch tor.State {
		case "downloading", "uploading", "stalledDL", "stalledUP", "metaDL",
			"forcedDL", "forcedUP", "checkingDL", "checkingUP", "moving", "allocating":
			active++
			paused = false
		case "error", "missingFiles":
			d.Issues++
			paused = false
			if tor.Name != "" {
				bad = append(bad, tor.Name)
			}
		case "pausedDL", "pausedUP":
		default:
			paused = false
		}
	}
	d.Paused = paused
	if len(tors) > 0 {
		d.Name = fmt.Sprintf("%d/%d active", active, len(tors))
	}
	if xfer.Dl > 0 || xfer.Ul > 0 {
		d.Speed = "dl " + rate(xfer.Dl) + " · ul " + rate(xfer.Ul)
	}
	if len(bad) > 0 {
		d.Status = "warning"
		d.Warning = strings.Join(bad[:min(2, len(bad))], ", ")
	}
	return d, nil
}

// proxmoxWidget reads cluster resources with an API token.
// apiKey is user@realm!tokenid=secret, with or without the PVEAPIToken= prefix.
type proxmoxWidget struct{}

func (proxmoxWidget) Meta() Type {
	return Type{
		Type:        "proxmox",
		Name:        "Proxmox",
		Description: "Running guests across the cluster",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "https://pve.local:8006", Required: true},
			{Key: "apiKey", Label: "API token", Placeholder: "user@pam!dash=secret", Secret: true, Required: true},
		},
	}
}

func (proxmoxWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint, err := endpointOf(cfg)
	if err != nil {
		return nil, err
	}
	token := cfgStr(cfg, "apiKey")
	if token == "" {
		return nil, fmt.Errorf("apiKey required")
	}
	if !strings.HasPrefix(token, "PVEAPIToken=") {
		token = "PVEAPIToken=" + token
	}
	code, _, body, err := call(ctx, http.MethodGet, endpoint+"/api2/json/cluster/resources", map[string]string{"Authorization": token}, nil, "")
	if err != nil {
		return nil, err
	}
	var res struct {
		Data []struct {
			Type   string `json:"type"`
			Status string `json:"status"`
			Name   string `json:"name"`
		} `json:"data"`
	}
	if err := okJSON(code, body, &res); err != nil {
		return nil, err
	}
	d := &svcData{Status: "online"}
	var badNodes []string
	for _, r := range res.Data {
		switch r.Type {
		case "qemu", "lxc":
			d.Total++
			if r.Status == "running" {
				d.Up++
			}
		case "node":
			if r.Status != "online" && r.Name != "" {
				badNodes = append(badNodes, r.Name)
			}
		}
	}
	if len(badNodes) > 0 {
		d.Status = "warning"
		d.Issues = len(badNodes)
		d.Warning = strings.Join(badNodes[:min(3, len(badNodes))], ", ")
	}
	return d, nil
}

// portainerWidget counts containers from endpoint snapshots, and falls back
// to the Docker container list when a snapshot has not been taken yet.
type portainerWidget struct{}

func (portainerWidget) Meta() Type {
	return Type{
		Type:        "portainer",
		Name:        "Portainer",
		Description: "Running and total containers",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://portainer.local:9000", Required: true},
			{Key: "apiKey", Label: "API key", Secret: true, Required: true},
		},
	}
}

func (portainerWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint, err := endpointOf(cfg)
	if err != nil {
		return nil, err
	}
	key := cfgStr(cfg, "apiKey")
	if key == "" {
		return nil, fmt.Errorf("apiKey required")
	}
	hdr := map[string]string{"X-API-Key": key}
	code, _, body, err := call(ctx, http.MethodGet, endpoint+"/api/endpoints", hdr, nil, "")
	if err != nil {
		return nil, err
	}
	var ends []struct {
		ID        int    `json:"Id"`
		Name      string `json:"Name"`
		Snapshots []struct {
			Docker struct {
				Running int `json:"RunningContainerCount"`
				Total   int `json:"ContainerCount"`
			} `json:"DockerSnapshot"`
		} `json:"Snapshots"`
	}
	if err := okJSON(code, body, &ends); err != nil {
		return nil, err
	}
	d := &svcData{Status: "online"}
	if len(ends) == 1 {
		d.Name = ends[0].Name
	}
	var plain []int
	for _, e := range ends {
		if len(e.Snapshots) == 0 {
			plain = append(plain, e.ID)
			continue
		}
		snap := e.Snapshots[len(e.Snapshots)-1].Docker
		d.Up += snap.Running
		d.Total += snap.Total
	}
	// A cold Portainer has no snapshot yet. Ask Docker directly, capped so a
	// big endpoint list cannot blow the widget timeout.
	if len(plain) > 4 {
		plain = plain[:4]
	}
	for _, id := range plain {
		code, _, body, err = call(ctx, http.MethodGet, fmt.Sprintf("%s/api/endpoints/%d/docker/containers/json?all=1", endpoint, id), hdr, nil, "")
		if err != nil {
			return nil, err
		}
		var boxes []struct {
			State string `json:"State"`
		}
		if err := okJSON(code, body, &boxes); err != nil {
			return nil, err
		}
		for _, b := range boxes {
			d.Total++
			if b.State == "running" {
				d.Up++
			}
		}
	}
	return d, nil
}

// npmWidget signs in to Nginx Proxy Manager and counts proxy hosts.
type npmWidget struct{}

func (npmWidget) Meta() Type {
	return Type{
		Type:        "nginxpm",
		Name:        "Nginx Proxy Manager",
		Description: "Enabled proxy hosts",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://npm.local:81", Required: true},
			{Key: "username", Label: "Email", Placeholder: "admin@example.com", Required: true},
			{Key: "apiKey", Label: "Password", Secret: true, Required: true},
		},
	}
}

func (npmWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint, err := endpointOf(cfg)
	if err != nil {
		return nil, err
	}
	email, pass := cfgStr(cfg, "username"), cfgStr(cfg, "apiKey")
	if email == "" || pass == "" {
		return nil, fmt.Errorf("email and password required")
	}
	raw, err := json.Marshal(map[string]string{"identity": email, "secret": pass})
	if err != nil {
		return nil, err
	}
	code, _, body, err := call(ctx, http.MethodPost, endpoint+"/api/tokens", nil, strings.NewReader(string(raw)), "application/json")
	if err != nil {
		return nil, err
	}
	var tok struct {
		Token string `json:"token"`
	}
	if err := okJSON(code, body, &tok); err != nil {
		return nil, err
	}
	if tok.Token == "" {
		return nil, fmt.Errorf("upstream returned no token")
	}
	code, _, body, err = call(ctx, http.MethodGet, endpoint+"/api/nginx/proxy-hosts", map[string]string{"Authorization": "Bearer " + tok.Token}, nil, "")
	if err != nil {
		return nil, err
	}
	var hosts []struct {
		Enabled bool `json:"enabled"`
	}
	if err := okJSON(code, body, &hosts); err != nil {
		return nil, err
	}
	d := &svcData{Status: "online", Total: len(hosts)}
	for _, h := range hosts {
		if h.Enabled {
			d.Up++
		}
	}
	return d, nil
}

// homeAssistantWidget counts entities and surfaces unavailable ones.
type homeAssistantWidget struct{}

func (homeAssistantWidget) Meta() Type {
	return Type{
		Type:        "homeassistant",
		Name:        "Home Assistant",
		Description: "Entities, and which are unavailable",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://homeassistant.local:8123", Required: true},
			{Key: "apiKey", Label: "Long-lived token", Secret: true, Required: true},
		},
	}
}

func (homeAssistantWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint, err := endpointOf(cfg)
	if err != nil {
		return nil, err
	}
	token := cfgStr(cfg, "apiKey")
	if token == "" {
		return nil, fmt.Errorf("apiKey required")
	}
	code, _, body, err := call(ctx, http.MethodGet, endpoint+"/api/states", map[string]string{"Authorization": "Bearer " + token}, nil, "")
	if err != nil {
		return nil, err
	}
	var states []struct {
		EntityID string `json:"entity_id"`
		State    string `json:"state"`
	}
	if err := okJSON(code, body, &states); err != nil {
		return nil, err
	}
	d := &svcData{Status: "online", Total: len(states)}
	var bad []string
	for _, s := range states {
		if s.State == "unavailable" || s.State == "unknown" {
			d.Issues++
			if s.EntityID != "" {
				bad = append(bad, s.EntityID)
			}
			continue
		}
		d.Up++
	}
	if d.Issues > 0 {
		d.Status = "warning"
		d.Warning = strings.Join(bad[:min(2, len(bad))], ", ")
	}
	return d, nil
}

// syncthingWidget reports connected devices and the daemon version.
type syncthingWidget struct{}

func (syncthingWidget) Meta() Type {
	return Type{
		Type:        "syncthing",
		Name:        "Syncthing",
		Description: "Connected devices",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://syncthing.local:8384", Required: true},
			{Key: "apiKey", Label: "API key", Secret: true, Required: true},
		},
	}
}

func (syncthingWidget) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	endpoint, err := endpointOf(cfg)
	if err != nil {
		return nil, err
	}
	key := cfgStr(cfg, "apiKey")
	if key == "" {
		return nil, fmt.Errorf("apiKey required")
	}
	hdr := map[string]string{"X-API-Key": key}
	code, _, body, err := call(ctx, http.MethodGet, endpoint+"/rest/system/version", hdr, nil, "")
	if err != nil {
		return nil, err
	}
	var ver struct {
		Version string `json:"version"`
	}
	if err := okJSON(code, body, &ver); err != nil {
		return nil, err
	}
	code, _, body, err = call(ctx, http.MethodGet, endpoint+"/rest/system/connections", hdr, nil, "")
	if err != nil {
		return nil, err
	}
	var con struct {
		Connections map[string]struct {
			Connected bool `json:"connected"`
		} `json:"connections"`
	}
	if err := okJSON(code, body, &con); err != nil {
		return nil, err
	}
	d := &svcData{Status: "online", Version: ver.Version, Total: len(con.Connections)}
	for _, c := range con.Connections {
		if c.Connected {
			d.Up++
		}
	}
	if d.Total > 0 && d.Up == 0 {
		d.Status = "warning"
		d.Warning = "no devices connected"
	}
	return d, nil
}
