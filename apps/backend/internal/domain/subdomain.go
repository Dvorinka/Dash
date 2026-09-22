package domain

// Subdomain discovery: certificate-transparency names from crt.sh, filtered
// to the owned suffix, then resolved via A/AAAA. Both network steps are
// injected seams so tests never leave the process.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Subdomain is one discovered hostname and its resolved addresses.
type Subdomain struct {
	Name string   `json:"name"`
	IPs  []string `json:"ips"`
}

// CTNames returns candidate hostnames for domain from crt.sh.
// Seam: tests substitute a stub.
var CTNames = func(ctx context.Context, domain string) ([]string, error) {
	url := "https://crt.sh/?q=%25." + domain + "&output=json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("crt.sh unreachable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crt.sh returned %d", resp.StatusCode)
	}
	// Bound the response — popular domains can return megabytes.
	var rows []struct {
		NameValue  string `json:"name_value"`
		CommonName string `json:"common_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&rows); err != nil {
		return nil, fmt.Errorf("crt.sh decode failed")
	}
	var out []string
	for _, r := range rows {
		out = append(out, r.NameValue, r.CommonName)
	}
	return out, nil
}

// ResolveIPs returns A/AAAA addresses for a hostname. Seam: tests stub it.
var ResolveIPs = func(ctx context.Context, name string) ([]string, error) {
	addrs, err := net.DefaultResolver.LookupIP(ctx, "ip", name)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a.String())
	}
	return out, nil
}

// FilterCTNames reduces raw cert names to owned, canonical hostnames:
// lowercase, wildcard prefix stripped, must equal domain or end in .domain.
// CT names are untrusted text — stored and rendered as text only.
func FilterCTNames(raw []string, domain string) []string {
	suffix := "." + domain
	seen := map[string]bool{}
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		for _, name := range strings.Split(r, "\n") {
			name = strings.ToLower(strings.TrimSpace(name))
			name = strings.TrimPrefix(name, "*.")
			if name == "" || (!strings.HasSuffix(name, suffix) && name != domain) {
				continue
			}
			if strings.ContainsAny(name, " \t/\\\"'<>@") {
				continue // cert names are untrusted; drop malformed entries
			}
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out
}

// DiscoverSubdomains queries CT and resolves each surviving name, bounded:
// at most 500 candidates considered, 100 resolved, 8 concurrent lookups.
func DiscoverSubdomains(ctx context.Context, domain string) ([]Subdomain, error) {
	raw, err := CTNames(ctx, domain)
	if err != nil {
		return nil, err
	}
	names := FilterCTNames(raw, domain)
	if len(names) > 500 {
		names = names[:500]
	}
	if len(names) > 100 {
		names = names[:100]
	}
	out := make([]Subdomain, len(names))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, name := range names {
		out[i].Name = name
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, name string) {
			defer wg.Done()
			defer func() { <-sem }()
			rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if ips, err := ResolveIPs(rctx, name); err == nil {
				out[i].IPs = ips
			}
		}(i, name)
	}
	wg.Wait()
	return out, nil
}
