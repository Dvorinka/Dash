// WHOIS data acquisition. Ported from Beszel's hub/domains/whois — the
// portable parts only: RDAP via IANA bootstrap, TCP WHOIS on :43, and the
// registry-format parser. Dropped: native `whois` exec (breaks the
// single-binary model), EURid/whois.com HTML scraping (fragile), and the
// paid WhoisXML API.
package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// WhoisData is the registry half of a Lookup result.
type WhoisData struct {
	DomainName       string
	Statuses         []string
	DNSSEC           string // "signed" | "unsigned" | ""
	ExpiryDate       *time.Time
	CreationDate     *time.Time
	UpdatedDate      *time.Time
	RegistrarName    string
	RegistrarID      string
	RegistrarURL     string
	RegistryDomainID string
	RegistrantName   string
	RegistrantOrg    string
	RegistrantCountry string
	AbuseEmail       string
	PrivacyEnabled   bool
	TransferLock     bool
}

// Whois looks up registration data: RDAP first (clean JSON), TCP WHOIS as
// fallback for TLDs without RDAP service.
func (l *Lookup) Whois(ctx context.Context, name string) (*WhoisData, error) {
	if d, err := l.rdap(ctx, name); err == nil && d.valid() {
		return d, nil
	}
	if d, err := tcpWhois(ctx, name); err == nil && d.valid() {
		return d, nil
	}
	return nil, fmt.Errorf("no whois data for %s", name)
}

func (d *WhoisData) valid() bool {
	return d != nil && (d.ExpiryDate != nil || d.CreationDate != nil ||
		(d.RegistrarName != "" && d.RegistrarName != "Unknown") || len(d.Statuses) > 0)
}

// --- RDAP ------------------------------------------------------------------

// rdapBases caches TLD -> RDAP base URL from the IANA bootstrap registry.
var rdapBases = struct {
	sync.Mutex
	m map[string]string
}{m: map[string]string{}}

func (l *Lookup) rdap(ctx context.Context, name string) (*WhoisData, error) {
	tld := tldOf(name)
	if tld == "" {
		return nil, fmt.Errorf("no tld")
	}
	base, err := rdapBase(ctx, tld)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/domain/"+name, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/rdap+json")
	resp, err := l.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rdap status %d", resp.StatusCode)
	}

	var r struct {
		LdhName string   `json:"ldhName"`
		Handle  string   `json:"handle"`
		Status  []string `json:"status"`
		Events  []struct {
			Action string `json:"eventAction"`
			Date   string `json:"eventDate"`
		} `json:"events"`
		Entities []struct {
			Roles     []string `json:"roles"`
			PublicIDs []struct {
				Type       string `json:"type"`
				Identifier string `json:"identifier"`
			} `json:"publicIds"`
			VCard []any `json:"vcardArray"`
		} `json:"entities"`
		SecureDNS struct {
			ZoneSigned bool `json:"zoneSigned"`
		} `json:"secureDNS"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil {
		return nil, err
	}

	d := &WhoisData{DomainName: r.LdhName, Statuses: r.Status, RegistryDomainID: r.Handle}
	if r.SecureDNS.ZoneSigned {
		d.DNSSEC = "signed"
	}
	for _, e := range r.Events {
		t, err := time.Parse(time.RFC3339, e.Date)
		if err != nil || t.IsZero() {
			continue
		}
		switch e.Action {
		case "registration":
			d.CreationDate = &t
		case "expiration":
			d.ExpiryDate = &t
		case "last changed":
			d.UpdatedDate = &t
		}
	}
	for _, ent := range r.Entities {
		if !hasRole(ent.Roles, "registrar") {
			continue
		}
		// vcardArray: ["vcard", [[name, params, type, value], ...]]
		if len(ent.VCard) > 1 {
			if props, ok := ent.VCard[1].([]any); ok {
				for _, p := range props {
					if arr, ok := p.([]any); ok && len(arr) >= 4 && arr[0] == "fn" {
						if s, ok := arr[3].(string); ok {
							d.RegistrarName = s
						}
					}
				}
			}
		}
		for _, pid := range ent.PublicIDs {
			if pid.Type == "IANA Registrar ID" {
				d.RegistrarID = pid.Identifier
			}
		}
	}
	d.derive()
	return d, nil
}

func hasRole(roles []string, want string) bool {
	for _, r := range roles {
		if r == want {
			return true
		}
	}
	return false
}

func rdapBase(ctx context.Context, tld string) (string, error) {
	rdapBases.Lock()
	defer rdapBases.Unlock()
	if u, ok := rdapBases.m[tld]; ok {
		return u, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://data.iana.org/rdap/dns.json", nil)
	if err != nil {
		return "", err
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var boot struct {
		Services [][]any `json:"services"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&boot); err != nil {
		return "", err
	}
	for _, svc := range boot.Services {
		if len(svc) < 2 {
			continue
		}
		tlds, _ := svc[0].([]any)
		urls, _ := svc[1].([]any)
		if len(urls) == 0 {
			continue
		}
		base, _ := urls[0].(string)
		for _, t := range tlds {
			if s, ok := t.(string); ok {
				rdapBases.m[s] = strings.TrimSuffix(base, "/")
			}
		}
	}
	if u, ok := rdapBases.m[tld]; ok {
		return u, nil
	}
	return "", fmt.Errorf("no rdap server for .%s", tld)
}

// --- TCP WHOIS (:43) --------------------------------------------------------

var whoisServers = map[string]string{
	"com": "whois.verisign-grs.com", "net": "whois.verisign-grs.com",
	"org": "whois.pir.org", "io": "whois.nic.io", "co": "whois.nic.co",
	"dev": "whois.nic.google", "app": "whois.nic.google", "xyz": "whois.nic.xyz",
	"info": "whois.afilias.net", "biz": "whois.biz", "us": "whois.nic.us",
	"uk": "whois.nic.uk", "de": "whois.denic.de", "fr": "whois.nic.fr",
	"eu": "whois.eu", "nl": "whois.domain-registry.nl", "ca": "whois.cira.ca",
	"au": "whois.auda.org.au", "me": "whois.nic.me", "tv": "whois.nic.tv",
	"cc": "whois.nic.cc", "name": "whois.nic.name", "pro": "whois.nic.pro",
	"cz": "whois.nic.cz", "sk": "whois.sk-nic.sk", "pl": "whois.dns.pl",
	"se": "whois.iis.se", "fi": "whois.fi", "no": "whois.norid.no",
	"ch": "whois.nic.ch", "at": "whois.nic.at", "be": "whois.dns.be",
	"it": "whois.nic.it", "es": "whois.nic.es", "pt": "whois.dns.pt",
	"jp": "whois.jprs.jp", "kr": "whois.kr", "in": "whois.registry.in",
	"br": "whois.registro.br", "za": "whois.registry.net.za",
	"cloud": "whois.nic.cloud", "online": "whois.nic.online",
	"site": "whois.nic.site", "store": "whois.nic.store", "tech": "whois.nic.tech",
	"ai": "whois.nic.ai", "gg": "whois.gg", "sh": "whois.nic.sh",
}

func tcpWhois(ctx context.Context, name string) (*WhoisData, error) {
	tld := tldOf(name)
	server, ok := whoisServers[tld]
	if !ok {
		server = "whois.iana.org"
	}
	d := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(server, "43"))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	if _, err := conn.Write([]byte(name + "\r\n")); err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(conn, 1<<20))
	if err != nil && len(raw) == 0 {
		return nil, err
	}
	out := parseWhois(string(raw), name)
	out.derive()
	return out, nil
}

// parseWhois flattens "Key: Value" lines and extracts the fields we store.
// Registry formats vary wildly; the field-name fallbacks cover the common set.
func parseWhois(raw, name string) *WhoisData {
	d := &WhoisData{DomainName: name}
	kv := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:idx]))
		key = strings.NewReplacer(" ", "_", "/", "_", "-", "_").Replace(key)
		val := strings.TrimSpace(line[idx+1:])
		if val == "" || strings.HasPrefix(strings.ToUpper(val), "REDACTED") {
			continue
		}
		if _, dup := kv[key]; !dup {
			kv[key] = val
		}
		switch key {
		case "domain_status", "status":
			if s := parseStatus(val); len(s) > 0 {
				d.Statuses = append(d.Statuses, s...)
			}
		}
	}

	d.ExpiryDate = parseDate(kv["registry_expiry_date"], kv["registrar_registration_expiration_date"],
		kv["expiry_date"], kv["expiration_time"], kv["expire"], kv["paid_until"],
		kv["expire_date"], kv["renewal_date"], kv["valid_until"], kv["expires"])
	d.CreationDate = parseDate(kv["creation_date"], kv["created_date"], kv["created"],
		kv["registration_time"], kv["registered_on"], kv["domain_registered"], kv["registered"])
	d.UpdatedDate = parseDate(kv["updated_date"], kv["last_updated"], kv["last_modified"],
		kv["modified_date"], kv["modified"], kv["changed"])

	for _, k := range []string{"registrar", "registrar_name", "sponsoring_registrar", "registrar_organization"} {
		if v := kv[k]; v != "" {
			d.RegistrarName = v
			break
		}
	}
	if d.RegistrarName == "" {
		d.RegistrarName = "Unknown"
	}
	d.RegistrarID = kv["registrar_iana_id"]
	d.RegistrarURL = kv["registrar_url"]
	d.RegistryDomainID = kv["registry_domain_id"]

	d.RegistrantName = firstOf(kv, "registrant_name", "registrant", "holder", "domain_holder")
	d.RegistrantOrg = firstOf(kv, "registrant_organization", "org", "organization", "holder_org")
	d.RegistrantCountry = firstOf(kv, "registrant_country", "country", "holder_country")
	d.AbuseEmail = kv["registrar_abuse_contact_email"]

	if v := strings.ToLower(firstOf(kv, "dnssec", "dnssec_signed", "signed_dnssec")); v == "signed" || v == "yes" || v == "true" || v == "signeddelegation" {
		d.DNSSEC = "signed"
	} else if v != "" {
		d.DNSSEC = "unsigned"
	}
	return d
}

// derive fills computed flags after whichever lookup path populated d.
func (d *WhoisData) derive() {
	low := strings.ToLower(d.RegistrantName + " " + d.RegistrantOrg)
	for _, marker := range []string{"redacted", "privacy", "whoisguard", "not disclosed", "hidden", "data protected", "gdpr"} {
		if strings.Contains(low, marker) {
			d.PrivacyEnabled = true
			break
		}
	}
	for _, s := range d.Statuses {
		if sl := strings.ToLower(s); strings.Contains(sl, "transferprohibited") {
			d.TransferLock = true
			break
		}
	}
}

func firstOf(kv map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := kv[k]; v != "" {
			return v
		}
	}
	return ""
}

var knownStatuses = []string{
	"clientDeleteProhibited", "clientHold", "clientRenewProhibited",
	"clientTransferProhibited", "clientUpdateProhibited",
	"serverDeleteProhibited", "serverHold", "serverRenewProhibited",
	"serverTransferProhibited", "serverUpdateProhibited",
	"inactive", "ok", "pendingCreate", "pendingDelete", "pendingRenew",
	"pendingRestore", "pendingTransfer", "pendingUpdate",
	"addPeriod", "autoRenewPeriod", "renewPeriod", "transferPeriod",
}

func parseStatus(s string) []string {
	s = strings.ToLower(s)
	var out []string
	for _, k := range knownStatuses {
		if strings.Contains(s, strings.ToLower(k)) {
			out = append(out, k)
		}
	}
	return out
}

var dateFormats = []string{
	time.RFC3339, "2006-01-02T15:04:05Z0700", "2006-01-02T15:04:05",
	"2006-01-02 15:04:05", "2006-01-02 15:04:05.0", "2006-01-02",
	"01/02/2006", "02/01/2006", "02.01.2006", "02-Jan-2006", "2-Jan-2006",
	"Jan 2 2006", "January 2 2006", "2 Jan 2006", "Jan 02 2006",
	"2006-01-02 15:04:05 MST", "Mon, 02 Jan 2006 15:04:05 MST",
	"20060102", "20060102150405",
}

func parseDate(candidates ...string) *time.Time {
	for _, s := range candidates {
		s = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(s, "before "), "after "))
		if s == "" || strings.Contains(strings.ToUpper(s), "REDACTED") {
			continue
		}
		// Trim trailing "(YYYY-MM-DD)" style annotations and zone offsets in parens.
		if i := strings.Index(s, "("); i > 0 {
			s = strings.TrimSpace(s[:i])
		}
		for _, f := range dateFormats {
			if t, err := time.Parse(f, s); err == nil && !t.IsZero() {
				return &t
			}
		}
	}
	return nil
}

// tldOf returns the last label; registrable-domain granularity is
// unnecessary here since we only feed it to server maps.
func tldOf(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return ""
	}
	return strings.ToLower(parts[len(parts)-1])
}

// whoisPattern guards against sending garbage to whois servers.
var whoisPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

func validDomainName(s string) bool {
	return len(s) <= 253 && whoisPattern.MatchString(s)
}
