// Package monitor holds the uptime-check domain: the Monitor model mirroring
// the monitors table, plus the check implementations the scheduler invokes.
// No DB access, no HTTP handlers — pure check logic.
package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Monitor mirrors one row of the monitors table.
type Monitor struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	URL           string    `json:"url"`
	Hostname      string    `json:"hostname"`
	Port          int       `json:"port"`
	Method        string    `json:"method"`
	Headers       string    `json:"headers"` // JSON object
	Body          string    `json:"body"`
	Keyword       string    `json:"keyword"`
	KeywordInvert bool      `json:"keywordInvert"`
	JSONQuery     string    `json:"jsonQuery"`
	Expected      string    `json:"expected"`
	DNSType       string    `json:"dnsType"`
	IntervalS     int       `json:"intervalS"`
	TimeoutS      int       `json:"timeoutS"`
	Retries       int       `json:"retries"`
	Active        bool      `json:"active"`
	Status        string    `json:"status"`
	PushToken     string    `json:"pushToken,omitempty"`
	Tags          []string  `json:"tags"`
	Notes         string    `json:"notes"`
	Position      float64   `json:"position"`
	LastCheck     *string   `json:"lastCheck"`
	CreatedAt     string    `json:"createdAt"`
}

// Result is the outcome of one check.
type Result struct {
	Up         bool
	PingMs     int
	Msg        string
	CertExpiry int // days until TLS cert expiry; -1 when not applicable
}

// Types lists the check kinds the add-monitor dialog offers.
func Types() []string {
	return []string{"http", "tcp", "ping", "dns", "keyword", "json", "push"}
}

// Check dispatches to the checker for m.Type. Retries happen in the caller.
func Check(ctx context.Context, m *Monitor) *Result {
	timeout := time.Duration(m.TimeoutS) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch m.Type {
	case "http", "keyword", "json":
		return checkHTTP(ctx, m)
	case "tcp":
		return checkTCP(ctx, m)
	case "ping":
		return checkPing(ctx, m)
	case "dns":
		return checkDNS(ctx, m)
	default:
		return &Result{Msg: "unknown monitor type: " + m.Type}
	}
}

func hostOf(m *Monitor) string {
	if m.Hostname != "" {
		return m.Hostname
	}
	return m.URL
}

func checkHTTP(ctx context.Context, m *Monitor) *Result {
	method := m.Method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if m.Body != "" {
		body = strings.NewReader(m.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, m.URL, body)
	if err != nil {
		return &Result{Msg: "bad url: " + m.URL}
	}
	var hdrs map[string]string
	if json.Unmarshal([]byte(m.Headers), &hdrs) == nil {
		for k, v := range hdrs {
			req.Header.Set(k, v)
		}
	}
	if m.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	ping := int(time.Since(start).Milliseconds())
	if err != nil {
		return &Result{Msg: err.Error()}
	}
	defer resp.Body.Close()

	res := &Result{PingMs: ping}
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		res.CertExpiry = int(time.Until(resp.TLS.PeerCertificates[0].NotAfter).Hours() / 24)
	} else {
		res.CertExpiry = -1
	}

	switch m.Type {
	case "keyword", "json":
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		if err != nil {
			res.Msg = "read body: " + err.Error()
			return res
		}
		if m.Type == "keyword" {
			found := strings.Contains(string(raw), m.Keyword)
			if m.KeywordInvert {
				found = !found
			}
			res.Up = found
			res.Msg = fmt.Sprintf("keyword %q %s", m.Keyword, map[bool]string{true: "found", false: "absent"}[found])
			return res
		}
		return checkJSONBody(res, raw, m)
	default:
		res.Up = resp.StatusCode < 400
		res.Msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return res
	}
}

// checkJSONBody walks a dot path ("a.b.0.c") into decoded JSON and compares
// the leaf to m.Expected.
func checkJSONBody(res *Result, raw []byte, m *Monitor) *Result {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		res.Msg = "invalid JSON: " + err.Error()
		return res
	}
	for _, key := range strings.Split(m.JSONQuery, ".") {
		switch n := v.(type) {
		case map[string]any:
			v = n[key]
		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(n) {
				res.Msg = "json path missing: " + m.JSONQuery
				return res
			}
			v = n[i]
		default:
			res.Msg = "json path missing: " + m.JSONQuery
			return res
		}
	}
	got := fmt.Sprint(v)
	res.Up = got == m.Expected
	res.Msg = fmt.Sprintf("%s = %q (want %q)", m.JSONQuery, got, m.Expected)
	return res
}

func checkTCP(ctx context.Context, m *Monitor) *Result {
	host := hostOf(m)
	if host == "" {
		return &Result{Msg: "hostname or url required"}
	}
	addr := net.JoinHostPort(host, strconv.Itoa(m.Port))
	start := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return &Result{Msg: err.Error()}
	}
	_ = conn.Close()
	return &Result{Up: true, PingMs: int(time.Since(start).Milliseconds()), Msg: "port open"}
}

// checkPing sends one ICMP echo. Raw sockets need CAP_NET_RAW; the udp4
// datagram path works unprivileged on Linux via ping_group_range.
func checkPing(ctx context.Context, m *Monitor) *Result {
	host := hostOf(m)
	if host == "" {
		return &Result{Msg: "hostname required"}
	}
	dst, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		if dst, err = net.ResolveIPAddr("ip6", host); err != nil {
			return &Result{Msg: "resolve: " + err.Error()}
		}
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		conn, err = icmp.ListenPacket("udp4", "0.0.0.0")
	}
	if err != nil {
		return &Result{Msg: "icmp unavailable (needs cap_net_raw or ping_group_range): " + err.Error()}
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(4 * time.Second))

	wm := icmp.Message{Type: ipv4.ICMPTypeEcho, Code: 0, Body: &icmp.Echo{
		ID:   1,
		Seq:  1,
		Data: []byte("dash"),
	}}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return &Result{Msg: err.Error()}
	}
	start := time.Now()
	var dstAddr net.Addr = dst
	if _, err := conn.WriteTo(wb, dstAddr); err != nil {
		return &Result{Msg: err.Error()}
	}
	rb := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFrom(rb)
		if err != nil {
			return &Result{Msg: "no reply: " + err.Error()}
		}
		rm, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), rb[:n])
		if err != nil {
			continue
		}
		if rm.Type == ipv4.ICMPTypeEchoReply {
			return &Result{Up: true, PingMs: int(time.Since(start).Milliseconds()), Msg: "echo reply"}
		}
	}
}

func checkDNS(ctx context.Context, m *Monitor) *Result {
	host := hostOf(m)
	if host == "" {
		return &Result{Msg: "hostname required"}
	}
	r := net.DefaultResolver
	start := time.Now()
	var answers []string
	var err error
	switch strings.ToUpper(m.DNSType) {
	case "AAAA":
		var ips []net.IP
		ips, err = r.LookupIP(ctx, "ip6", host)
		for _, ip := range ips {
			answers = append(answers, ip.String())
		}
	case "MX":
		var mxs []*net.MX
		mxs, err = r.LookupMX(ctx, host)
		for _, mx := range mxs {
			answers = append(answers, mx.Host)
		}
	case "NS":
		var nss []*net.NS
		nss, err = r.LookupNS(ctx, host)
		for _, ns := range nss {
			answers = append(answers, ns.Host)
		}
	case "TXT":
		answers, err = r.LookupTXT(ctx, host)
	case "CNAME":
		var c string
		c, err = r.LookupCNAME(ctx, host)
		if c != "" {
			answers = []string{c}
		}
	default: // A
		var ips []net.IP
		ips, err = r.LookupIP(ctx, "ip4", host)
		for _, ip := range ips {
			answers = append(answers, ip.String())
		}
	}
	ping := int(time.Since(start).Milliseconds())
	if err != nil {
		return &Result{Msg: err.Error()}
	}
	if len(answers) == 0 {
		return &Result{Msg: "no records"}
	}
	res := &Result{Up: true, PingMs: ping, Msg: strings.Join(answers, ", ")}
	if m.Expected != "" {
		for _, a := range answers {
			if strings.Contains(a, m.Expected) {
				return res
			}
		}
		res.Up = false
		res.Msg = fmt.Sprintf("no record contains %q (got %s)", m.Expected, res.Msg)
	}
	return res
}
