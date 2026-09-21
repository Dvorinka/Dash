// Package domain is the domain-intelligence half of the Beszel/Domain-Locker
// merge: given a domain name it gathers WHOIS/RDAP registration data, DNS
// records, TLS certificate details, host geolocation, and provider hints.
// Pure lookup logic — no DB, no HTTP handlers.
package domain

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Lookup bundles the shared HTTP client used by the network probes.
type Lookup struct {
	http *http.Client
}

func NewLookup() *Lookup {
	return &Lookup{http: &http.Client{Timeout: 10 * time.Second}}
}

// Result is everything one refresh learns about a domain. Fields stay flat —
// the API layer maps them 1:1 onto the domains table columns.
type Result struct {
	Name string `json:"name"`
	TLD  string `json:"tld"`

	// WHOIS / RDAP
	ExpiryDate       *time.Time `json:"expiryDate"`
	CreationDate     *time.Time `json:"creationDate"`
	UpdatedDate      *time.Time `json:"updatedDate"`
	RegistrarName    string     `json:"registrarName"`
	RegistrarID      string     `json:"registrarId"`
	RegistrarURL     string     `json:"registrarUrl"`
	RegistryDomainID string     `json:"registryDomainId"`
	DNSSEC           string     `json:"dnssec"`
	Statuses         []string   `json:"statuses"`
	PrivacyEnabled   bool       `json:"privacyEnabled"`
	TransferLock     bool       `json:"transferLock"`
	RegistrantName   string     `json:"registrantName"`
	RegistrantOrg    string     `json:"registrantOrg"`
	RegistrantCountry string    `json:"registrantCountry"`
	AbuseEmail       string     `json:"abuseEmail"`

	// DNS
	NameServers []string `json:"nameServers"`
	MXRecords   []string `json:"mxRecords"`
	TXTRecords  []string `json:"txtRecords"`
	CNAME       string   `json:"cname"`
	IPv4        []string `json:"ipv4"`
	IPv6        []string `json:"ipv6"`

	// TLS leaf certificate
	SSLIssuer       string     `json:"sslIssuer"`
	SSLValidFrom    *time.Time `json:"sslValidFrom"`
	SSLValidTo      *time.Time `json:"sslValidTo"`
	SSLSubject      string     `json:"sslSubject"`
	SSLFingerprint  string     `json:"sslFingerprint"`
	SSLKeySize      int        `json:"sslKeySize"`
	SSLSigAlgo      string     `json:"sslSigAlgo"`
	SSLAltNames     []string   `json:"sslAltNames"`

	// Host geolocation (ip-api.com; empty when unreachable/private)
	HostCountry     string  `json:"hostCountry"`
	HostCountryCode string  `json:"hostCountryCode"`
	HostRegion      string  `json:"hostRegion"`
	HostCity        string  `json:"hostCity"`
	HostISP         string  `json:"hostIsp"`
	HostOrg         string  `json:"hostOrg"`
	HostAS          string  `json:"hostAs"`
	HostLat         float64 `json:"hostLat"`
	HostLon         float64 `json:"hostLon"`

	// Provider detection
	DNSProvider     string `json:"dnsProvider"`
	EmailProvider   string `json:"emailProvider"`
	HostingProvider string `json:"hostingProvider"`
	CAProvider      string `json:"caProvider"`

	Headers   map[string]string `json:"headers"`
	FaviconURL string           `json:"faviconUrl"`
	Error     string            `json:"error"` // non-fatal partial failure note
}

// Clean normalises user input to a bare registrable hostname.
func Clean(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = regexp.MustCompile(`^https?://`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`^www\.`).ReplaceAllString(name, "")
	if i := strings.IndexAny(name, "/?#"); i >= 0 {
		name = name[:i]
	}
	if i := strings.Index(name, ":"); i >= 0 {
		name = name[:i]
	}
	return name
}

// Valid reports whether s is a plausible DNS name we can query for.
func Valid(s string) bool { return validDomainName(s) }

// Run gathers everything. Individual probe failures are non-fatal — a domain
// with no HTTPS still returns WHOIS+DNS. The returned Error field carries the
// first fatal WHOIS failure when even registration data is unavailable.
func (l *Lookup) Run(ctx context.Context, rawName string) *Result {
	name := Clean(rawName)
	r := &Result{
		Name:        name,
		TLD:         tldOf(name),
		Statuses:    []string{},
		NameServers: []string{},
		MXRecords:   []string{},
		TXTRecords:  []string{},
		IPv4:        []string{},
		IPv6:        []string{},
		SSLAltNames: []string{},
		Headers:     map[string]string{},
		FaviconURL:  fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=128", name),
	}

	if w, err := l.Whois(ctx, name); err == nil {
		r.ExpiryDate = w.ExpiryDate
		r.CreationDate = w.CreationDate
		r.UpdatedDate = w.UpdatedDate
		r.RegistrarName = w.RegistrarName
		r.RegistrarID = w.RegistrarID
		r.RegistrarURL = w.RegistrarURL
		r.RegistryDomainID = w.RegistryDomainID
		r.DNSSEC = w.DNSSEC
		r.Statuses = w.Statuses
		r.PrivacyEnabled = w.PrivacyEnabled
		r.TransferLock = w.TransferLock
		r.RegistrantName = w.RegistrantName
		r.RegistrantOrg = w.RegistrantOrg
		r.RegistrantCountry = w.RegistrantCountry
		r.AbuseEmail = w.AbuseEmail
	} else {
		r.Error = "whois: " + err.Error()
	}

	l.dns(ctx, name, r)
	l.cert(ctx, name, r)
	if len(r.IPv4) > 0 {
		l.host(r.IPv4[0], r)
	}
	l.headers(ctx, name, r)

	r.DNSProvider = detectDNSProvider(r.NameServers)
	r.EmailProvider = detectEmailProvider(r.MXRecords)
	r.HostingProvider = detectHostingProvider(r.Headers)
	r.CAProvider = detectCA(r.SSLIssuer)
	return r
}

// --- DNS --------------------------------------------------------------------

func (l *Lookup) dns(ctx context.Context, name string, r *Result) {
	res := net.DefaultResolver
	if ns, err := res.LookupNS(ctx, name); err == nil {
		for _, n := range ns {
			r.NameServers = append(r.NameServers, strings.TrimSuffix(strings.ToLower(n.Host), "."))
		}
	}
	if mx, err := res.LookupMX(ctx, name); err == nil {
		for _, m := range mx {
			r.MXRecords = append(r.MXRecords, fmt.Sprintf("%s (pri %d)", strings.TrimSuffix(m.Host, "."), m.Pref))
		}
	}
	if txt, err := res.LookupTXT(ctx, name); err == nil {
		r.TXTRecords = txt
	}
	if c, err := res.LookupCNAME(ctx, name); err == nil && c != "" && strings.TrimSuffix(c, ".") != name {
		r.CNAME = strings.TrimSuffix(c, ".")
	}
	if ips, err := res.LookupIP(ctx, "ip4", name); err == nil {
		for _, ip := range ips {
			r.IPv4 = append(r.IPv4, ip.String())
		}
	}
	if ips, err := res.LookupIP(ctx, "ip6", name); err == nil {
		for _, ip := range ips {
			r.IPv6 = append(r.IPv6, ip.String())
		}
	}
}

// --- TLS leaf certificate ----------------------------------------------------

func (l *Lookup) cert(ctx context.Context, name string, r *Result) {
	d := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := tls.DialWithDialer(d, "tcp", net.JoinHostPort(name, "443"), &tls.Config{
		ServerName:         name,
		InsecureSkipVerify: true, // inspecting, not authenticating — expiry is data
	})
	if err != nil {
		return
	}
	defer conn.Close()
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return
	}
	c := certs[0]
	if len(c.Issuer.Organization) > 0 {
		r.SSLIssuer = c.Issuer.Organization[0]
	} else {
		r.SSLIssuer = c.Issuer.CommonName
	}
	r.SSLValidFrom = &c.NotBefore
	r.SSLValidTo = &c.NotAfter
	r.SSLSubject = c.Subject.CommonName
	fp := sha256.Sum256(c.Raw)
	h := strings.ToUpper(hex.EncodeToString(fp[:]))
	var parts []string
	for i := 0; i+2 <= len(h); i += 2 {
		parts = append(parts, h[i:i+2])
	}
	r.SSLFingerprint = strings.Join(parts, ":")
	r.SSLSigAlgo = c.SignatureAlgorithm.String()
	switch k := c.PublicKey.(type) {
	case *rsa.PublicKey:
		r.SSLKeySize = k.N.BitLen()
	case *ecdsa.PublicKey:
		r.SSLKeySize = k.Curve.Params().BitSize
	}
	for _, n := range c.DNSNames {
		r.SSLAltNames = append(r.SSLAltNames, n)
	}
}

// --- Host geolocation (ip-api.com, free non-commercial tier) -----------------

func (l *Lookup) host(ip string, r *Result) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,regionName,city,lat,lon,isp,org,as", ip), nil)
	if err != nil {
		return
	}
	resp, err := l.http.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var g struct {
		Status  string  `json:"status"`
		Country string  `json:"country"`
		Code    string  `json:"countryCode"`
		Region  string  `json:"regionName"`
		City    string  `json:"city"`
		Lat     float64 `json:"lat"`
		Lon     float64 `json:"lon"`
		ISP     string  `json:"isp"`
		Org     string  `json:"org"`
		AS      string  `json:"as"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&g); err != nil || g.Status != "success" {
		return
	}
	r.HostCountry, r.HostCountryCode = g.Country, g.Code
	r.HostRegion, r.HostCity = g.Region, g.City
	r.HostISP, r.HostOrg, r.HostAS = g.ISP, g.Org, g.AS
	r.HostLat, r.HostLon = g.Lat, g.Lon
}

// --- HTTP headers (feed provider detection) -----------------------------------

func (l *Lookup) headers(ctx context.Context, name string, r *Result) {
	for _, scheme := range []string{"https", "http"} {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, scheme+"://"+name, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Dash domain lookup")
		resp, err := l.http.Do(req)
		if err != nil {
			continue
		}
		for k, v := range resp.Header {
			if len(v) > 0 {
				r.Headers[strings.ToLower(k)] = v[0]
			}
		}
		resp.Body.Close()
		return
	}
}

// --- Provider detection (condensed port of beszel's detect package) -----------

func detectDNSProvider(ns []string) string {
	table := []struct{ sub, name string }{
		{"cloudflare", "Cloudflare"}, {"awsdns", "Amazon Route 53"},
		{"googledomains", "Google Domains"}, {"google.com", "Google Domains"},
		{"namecheap", "Namecheap"}, {"godaddy", "GoDaddy"}, {"domaincontrol", "GoDaddy"},
		{"nsone.net", "NS1"}, {"digitalocean", "DigitalOcean"}, {"linode", "Linode"},
		{"vultr", "Vultr"}, {"he.net", "Hurricane Electric"}, {"dynect", "Dyn (Oracle)"},
		{"ultradns", "UltraDNS"}, {"dnsimple", "DNSimple"}, {"hover", "Hover"},
		{"enom", "eNom"}, {"worldnic", "Network Solutions"}, {"zoneedit", "ZoneEdit"},
		{"easydns", "EasyDNS"}, {"gandi", "Gandi"}, {"ovh", "OVH"},
		{"hetzner", "Hetzner"}, {"azure-dns", "Microsoft Azure"},
		{"dns.google", "Google Cloud DNS"}, {"porkbun", "Porkbun"},
		{"cloudns", "ClouDNS"}, {"desec", "deSEC"}, {"nic.cz", "CZ.NIC"},
	}
	for _, n := range ns {
		n = strings.ToLower(n)
		for _, t := range table {
			if strings.Contains(n, t.sub) {
				return t.name
			}
		}
	}
	return ""
}

func detectEmailProvider(mx []string) string {
	table := []struct{ sub, name string }{
		{"google.com", "Google Workspace"}, {"aspmx", "Google Workspace"},
		{"outlook.com", "Microsoft 365"}, {"protection.outlook", "Microsoft 365"},
		{"mail.protection", "Microsoft 365"}, {"protonmail", "Proton Mail"},
		{"proton.me", "Proton Mail"}, {"zoho", "Zoho Mail"},
		{"fastmail", "Fastmail"}, {"mimecast", "Mimecast"},
		{"proofpoint", "Proofpoint"}, {"barracuda", "Barracuda"},
		{"mailgun", "Mailgun"}, {"sendgrid", "SendGrid"},
		{"amazonses", "Amazon SES"}, {"yandex", "Yandex"},
		{"tuta", "Tuta"}, {"mailbox.org", "mailbox.org"},
	}
	for _, m := range mx {
		m = strings.ToLower(m)
		for _, t := range table {
			if strings.Contains(m, t.sub) {
				return t.name
			}
		}
	}
	return ""
}

func detectHostingProvider(h map[string]string) string {
	server := strings.ToLower(h["server"] + " " + h["x-powered-by"] + " " + h["via"])
	if _, ok := h["cf-ray"]; ok || strings.Contains(server, "cloudflare") {
		return "Cloudflare"
	}
	if _, ok := h["x-vercel-id"]; ok || strings.Contains(server, "vercel") {
		return "Vercel"
	}
	if _, ok := h["x-nf-request-id"]; ok || strings.Contains(server, "netlify") {
		return "Netlify"
	}
	if _, ok := h["x-amz-cf-id"]; ok || strings.Contains(server, "cloudfront") {
		return "AWS CloudFront"
	}
	if _, ok := h["x-github-request-id"]; ok || strings.Contains(server, "github") {
		return "GitHub Pages"
	}
	if strings.Contains(server, "fly.io") || h["fly-request-id"] != "" {
		return "Fly.io"
	}
	if strings.Contains(server, "render") {
		return "Render"
	}
	table := []struct{ sub, name string }{
		{"nginx", "nginx"}, {"apache", "Apache"}, {"caddy", "Caddy"},
		{"litespeed", "LiteSpeed"}, {"openresty", "OpenResty"},
		{"envoy", "Envoy"}, {"awselb", "AWS ELB"}, {"gws", "Google"},
		{"kestrel", "Kestrel/.NET"}, {"cowboy", "Cowboy/Erlang"},
	}
	for _, t := range table {
		if strings.Contains(server, t.sub) {
			return t.name
		}
	}
	return ""
}

func detectCA(issuer string) string {
	i := strings.ToLower(issuer)
	table := []struct{ sub, name string }{
		{"let's encrypt", "Let's Encrypt"}, {"letsencrypt", "Let's Encrypt"},
		{"digicert", "DigiCert"}, {"sectigo", "Sectigo"}, {"comodoca", "Sectigo"},
		{"globalsign", "GlobalSign"}, {"godaddy", "GoDaddy"}, {"starfield", "GoDaddy"},
		{"entrust", "Entrust"}, {"google trust", "Google Trust Services"},
		{"amazon", "Amazon"}, {"zerossl", "ZeroSSL"}, {"buypass", "Buypass"},
		{"ssl.com", "SSL.com"}, {"actalis", "Actalis"}, {"certum", "Certum"},
		{"harica", "HARICA"}, {"swisssign", "SwissSign"}, {"cloudflare", "Cloudflare"},
		{"pki.goog", "Google Trust Services"}, {"microsoft", "Microsoft"},
	}
	for _, t := range table {
		if strings.Contains(i, t.sub) {
			return t.name
		}
	}
	return ""
}
