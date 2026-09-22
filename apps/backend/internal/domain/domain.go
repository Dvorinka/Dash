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

	// Structured intel — domainstack-style record list + vendor strip.
	Records   []DNSRecord `json:"records"`
	Providers Providers   `json:"providers"`

	Headers   map[string]string `json:"headers"`
	FaviconURL string           `json:"faviconUrl"`
	Error     string            `json:"error"` // non-fatal partial failure note

	mxPairs []mxPair // structured MX; MXRecords keeps the display string
}

// Provider names the vendor behind a record or role. Domain feeds the
// favicon icon rendered next to the name in the UI.
type Provider struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
	Icon   string `json:"icon"`
}

// Providers is the detected-vendor strip: registrar, DNS host, hosting,
// email, and certificate authority.
type Providers struct {
	Registrar *Provider `json:"registrar,omitempty"`
	DNS       *Provider `json:"dns,omitempty"`
	Email     *Provider `json:"email,omitempty"`
	Hosting   *Provider `json:"hosting,omitempty"`
	CA        *Provider `json:"ca,omitempty"`
}

// DNSRecord is one resolved record with the vendor its value points at.
type DNSRecord struct {
	Type     string    `json:"type"` // A AAAA CNAME NS MX TXT
	Value    string    `json:"value"`
	Priority *int      `json:"priority,omitempty"` // MX preference
	Provider *Provider `json:"provider,omitempty"`
}

type mxPair struct {
	host string
	pref int
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
		Records:     []DNSRecord{},
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
	l.identify(r)
	return r
}

// identify resolves provider vendors after every probe has run, then builds
// the structured record list the detail page renders.
func (l *Lookup) identify(r *Result) {
	r.Providers.Registrar = matchProvider(r.RegistrarName, registrarProviders)
	for _, n := range r.NameServers {
		if p := matchProvider(n, nsProviders); p != nil {
			r.Providers.DNS = p
			break
		}
	}
	for _, m := range r.mxPairs {
		if p := matchProvider(m.host, mailProviders); p != nil {
			r.Providers.Email = p
			break
		}
	}
	r.Providers.Hosting = detectHostProvider(r)
	r.Providers.CA = matchProvider(r.SSLIssuer, caProviders)
	// Legacy flat names mirror the strip for the list page + old clients.
	r.DNSProvider = nameOf(r.Providers.DNS)
	r.EmailProvider = nameOf(r.Providers.Email)
	r.HostingProvider = nameOf(r.Providers.Hosting)
	r.CAProvider = nameOf(r.Providers.CA)

	recs := make([]DNSRecord, 0,
		len(r.IPv4)+len(r.IPv6)+len(r.NameServers)+len(r.mxPairs)+len(r.TXTRecords)+1)
	add := func(t, v string, pri *int, p *Provider) {
		recs = append(recs, DNSRecord{Type: t, Value: v, Priority: pri, Provider: p})
	}
	for _, ip := range r.IPv4 {
		add("A", ip, nil, r.Providers.Hosting)
	}
	for _, ip := range r.IPv6 {
		add("AAAA", ip, nil, r.Providers.Hosting)
	}
	if r.CNAME != "" {
		add("CNAME", r.CNAME, nil, matchProvider(r.CNAME, hostTargets))
	}
	for _, n := range r.NameServers {
		add("NS", n, nil, matchProvider(n, nsProviders))
	}
	for _, m := range r.mxPairs {
		pri := m.pref
		add("MX", m.host, &pri, matchProvider(m.host, mailProviders))
	}
	for _, t := range r.TXTRecords {
		add("TXT", t, nil, matchTXTProvider(t))
	}
	r.Records = recs
}

func nameOf(p *Provider) string {
	if p == nil {
		return ""
	}
	return p.Name
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
			host := strings.TrimSuffix(strings.ToLower(m.Host), ".")
			r.mxPairs = append(r.mxPairs, mxPair{host: host, pref: int(m.Pref)})
			r.MXRecords = append(r.MXRecords, fmt.Sprintf("%s (pri %d)", host, m.Pref))
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
//
// Tables map a substring of the record value / org / issuer to a vendor name
// plus the brand domain used for the favicon icon. Order matters — first
// match wins, so specific substrings precede generic ones.

type provMatch struct{ sub, name, dom string }

func prov(name, dom string) *Provider {
	return &Provider{Name: name, Domain: dom,
		Icon: fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=64", dom)}
}

func matchProvider(haystack string, table []provMatch) *Provider {
	h := strings.ToLower(haystack)
	for _, t := range table {
		if strings.Contains(h, t.sub) {
			return prov(t.name, t.dom)
		}
	}
	return nil
}

var nsProviders = []provMatch{
	{"cloudflare", "Cloudflare", "cloudflare.com"},
	{"awsdns", "Amazon Route 53", "aws.amazon.com"},
	{"googledomains", "Google Domains", "domains.google"},
	{"dns.google", "Google Cloud DNS", "cloud.google.com"},
	{"azure-dns", "Microsoft Azure", "azure.microsoft.com"},
	{"domaincontrol", "GoDaddy", "godaddy.com"},
	{"godaddy", "GoDaddy", "godaddy.com"},
	{"registrar-servers", "Namecheap", "namecheap.com"},
	{"namecheap", "Namecheap", "namecheap.com"},
	{"nsone.net", "NS1", "ns1.com"},
	{"digitalocean", "DigitalOcean", "digitalocean.com"},
	{"linode", "Akamai", "akamai.com"},
	{"akamaitech", "Akamai", "akamai.com"},
	{"vultr", "Vultr", "vultr.com"},
	{"he.net", "Hurricane Electric", "he.net"},
	{"dynect", "Dyn (Oracle)", "dyn.com"},
	{"ultradns", "UltraDNS", "verisign.com"},
	{"dnsimple", "DNSimple", "dnsimple.com"},
	{"hover", "Hover", "hover.com"},
	{"enom", "eNom", "enom.com"},
	{"worldnic", "Network Solutions", "networksolutions.com"},
	{"zoneedit", "ZoneEdit", "zoneedit.com"},
	{"easydns", "EasyDNS", "easydns.com"},
	{"gandi", "Gandi", "gandi.net"},
	{"ovh", "OVHcloud", "ovhcloud.com"},
	{"hetzner", "Hetzner", "hetzner.com"},
	{"your-server.de", "Hetzner", "hetzner.com"},
	{"porkbun", "Porkbun", "porkbun.com"},
	{"cloudns", "ClouDNS", "cloudns.net"},
	{"desec", "deSEC", "desec.io"},
	{"nic.cz", "CZ.NIC", "nic.cz"},
	{"dnsmadeeasy", "DNS Made Easy", "dnsmadeeasy.com"},
	{"ui-dns", "IONOS", "ionos.com"},
	{"squarespacedns", "Squarespace", "squarespace.com"},
	{"wordpress.com", "WordPress.com", "wordpress.com"},
	{"dns-parking.com", "Hostinger", "hostinger.com"},
	{"hostinger", "Hostinger", "hostinger.com"},
	{"siteground", "SiteGround", "siteground.com"},
	{"dreamhost", "DreamHost", "dreamhost.com"},
	{"inmotionhosting", "InMotion", "inmotionhosting.com"},
	{"bluehost", "Bluehost", "bluehost.com"},
	{"hostgator", "HostGator", "hostgator.com"},
	{"kinstadns", "Kinsta", "kinsta.com"},
	{"kasserver", "ALL-INKL.COM", "all-inkl.com"},
	{"all-inkl", "ALL-INKL.COM", "all-inkl.com"},
	{"strato", "STRATO", "strato.de"},
	{"udagdns", "United Domains", "uniteddomains.com"},
	{"udag.de", "United Domains", "uniteddomains.com"},
	{"transip", "TransIP", "transip.nl"},
	{"infomaniak", "Infomaniak", "infomaniak.com"},
	{"one.com", "One.com", "one.com"},
	{"aruba", "Aruba", "aruba.it"},
	{"technorail", "Aruba", "aruba.it"},
	{"inwx", "INWX", "inwx.com"},
	{"kasserver.com", "ALL-INKL.COM", "all-inkl.com"},
	{"contabo", "Contabo", "contabo.com"},
	{"netcup", "netcup", "netcup.eu"},
	{"glesys", "GleSYS", "glesys.se"},
	{"scaleway", "Scaleway", "scaleway.com"},
	{"exoscale", "Exoscale", "exoscale.com"},
	{"dnspod", "DNSPod", "dnspod.com"},
	{"hicloud", "Huawei Cloud", "huaweicloud.com"},
	{"alidns", "Alibaba Cloud", "alibabacloud.com"},
	{"purelymail", "Purelymail", "purelymail.com"},
	{"migadu", "Migadu", "migadu.com"},
	{"improvmx", "ImprovMX", "improvmx.com"},
	{"bunny", "Bunny", "bunny.net"},
	{"edgekey", "Akamai", "akamai.com"},
	{"edgesuite", "Akamai", "akamai.com"},
	{"fastly", "Fastly", "fastly.com"},
	{"cdn77", "CDN77", "cdn77.com"},
	{"stackpath", "StackPath", "stackpath.com"},
	{"ns.cloudflare", "Cloudflare", "cloudflare.com"},
	{"vercel-dns", "Vercel", "vercel.com"},
}

var mailProviders = []provMatch{
	{"aspmx", "Google Workspace", "google.com"},
	{"googlemail", "Google Workspace", "google.com"},
	{"google.com", "Google Workspace", "google.com"},
	{"outlook.com", "Microsoft 365", "microsoft.com"},
	{"protection.outlook", "Microsoft 365", "microsoft.com"},
	{"mail.protection", "Microsoft 365", "microsoft.com"},
	{"protonmail", "Proton Mail", "proton.me"},
	{"proton.me", "Proton Mail", "proton.me"},
	{"zoho", "Zoho Mail", "zoho.com"},
	{"fastmail", "Fastmail", "fastmail.com"},
	{"messagingengine", "Fastmail", "fastmail.com"},
	{"mimecast", "Mimecast", "mimecast.com"},
	{"proofpoint", "Proofpoint", "proofpoint.com"},
	{"barracuda", "Barracuda", "barracuda.com"},
	{"mailgun", "Mailgun", "mailgun.com"},
	{"sendgrid", "SendGrid", "sendgrid.com"},
	{"amazonses", "Amazon SES", "aws.amazon.com"},
	{"yandex", "Yandex", "yandex.com"},
	{"tuta", "Tuta", "tuta.com"},
	{"mailbox.org", "mailbox.org", "mailbox.org"},
	{"purelymail", "Purelymail", "purelymail.com"},
	{"migadu", "Migadu", "migadu.com"},
	{"mxroute", "MXroute", "mxroute.com"},
	{"mymxroute", "MXroute", "mxroute.com"},
	{"icloud.com", "iCloud Mail", "apple.com"},
	{"infomaniak", "Infomaniak", "infomaniak.com"},
	{"mailpod", "One.com", "one.com"},
	{"one.com", "One.com", "one.com"},
	{"gmx", "GMX", "gmx.net"},
	{"forwardemail", "Forward Email", "forwardemail.net"},
	{"posteo", "Posteo", "posteo.de"},
	{"mailo", "Mailo", "mailo.com"},
	{"hey.com", "HEY", "hey.com"},
	{"privateemail", "Namecheap Private Email", "namecheap.com"},
	{"registrar-servers", "Namecheap", "namecheap.com"},
	{"ovh.net", "OVHcloud Mail", "ovhcloud.com"},
	{"perfora.net", "IONOS Mail", "ionos.com"},
	{"ui-dns", "IONOS", "ionos.com"},
	{"improvmx", "ImprovMX", "improvmx.com"},
	{"porkbun", "Porkbun Mail", "porkbun.com"},
	{"kundenserver", "IONOS Mail", "ionos.com"},
	{"t-online", "T-Online", "telekom.de"},
	{"web.de", "WEB.DE", "web.de"},
	{"mail.ru", "Mail.ru", "mail.ru"},
	{"qq.com", "QQ Mail", "qq.com"},
	{"163.com", "NetEase Mail", "163.com"},
	{"exim.rs", "Exim", "exim.org"},
	{"mx.cloudflare", "Cloudflare Email Routing", "cloudflare.com"},
}

// hostTargets: CNAME targets that reveal the hosting platform.
var hostTargets = []provMatch{
	{"github.io", "GitHub Pages", "github.com"},
	{"vercel-dns", "Vercel", "vercel.com"},
	{"vercel.app", "Vercel", "vercel.com"},
	{"netlify", "Netlify", "netlify.com"},
	{"cloudfront.net", "AWS CloudFront", "aws.amazon.com"},
	{"azurefd.net", "Azure Front Door", "azure.microsoft.com"},
	{"azureedge.net", "Azure CDN", "azure.microsoft.com"},
	{"cloudapp.net", "Microsoft Azure", "azure.microsoft.com"},
	{"trafficmanager.net", "Microsoft Azure", "azure.microsoft.com"},
	{"herokuapp.com", "Heroku", "heroku.com"},
	{"herokudns.com", "Heroku", "heroku.com"},
	{"workers.dev", "Cloudflare Workers", "cloudflare.com"},
	{"pages.dev", "Cloudflare Pages", "cloudflare.com"},
	{"cdn.cloudflare", "Cloudflare", "cloudflare.com"},
	{"fly.dev", "Fly.io", "fly.io"},
	{"onrender.com", "Render", "render.com"},
	{"railway.app", "Railway", "railway.app"},
	{"surge.sh", "Surge", "surge.sh"},
	{"neocities.org", "Neocities", "neocities.org"},
	{"readthedocs", "Read the Docs", "readthedocs.io"},
	{"webflow.io", "Webflow", "webflow.com"},
	{"squarespace.com", "Squarespace", "squarespace.com"},
	{"wixdns.net", "Wix", "wix.com"},
	{"wixsite.com", "Wix", "wix.com"},
	{"editorx.com", "Wix Studio", "wix.com"},
	{"myshopify.com", "Shopify", "shopify.com"},
	{"wpengine", "WP Engine", "wpengine.com"},
	{"wordpress.com", "WordPress.com", "wordpress.com"},
	{"googlehosted.com", "Google", "google.com"},
	{"googleapis.com", "Google Cloud", "cloud.google.com"},
	{"fastly.net", "Fastly", "fastly.com"},
	{"fastlylb.net", "Fastly", "fastly.com"},
	{"akamaiedge.net", "Akamai", "akamai.com"},
	{"edgekey.net", "Akamai", "akamai.com"},
	{"edgesuite.net", "Akamai", "akamai.com"},
	{"akamai.net", "Akamai", "akamai.com"},
	{"b-cdn.net", "Bunny", "bunny.net"},
	{"bunnycdn", "Bunny", "bunny.net"},
	{"elb.amazonaws.com", "AWS ELB", "aws.amazon.com"},
	{"amazonaws.com", "Amazon AWS", "aws.amazon.com"},
	{"stackpathdns", "StackPath", "stackpath.com"},
	{"kinsta.cloud", "Kinsta", "kinsta.com"},
	{"pantheonsite.io", "Pantheon", "pantheon.io"},
	{"pantheon.io", "Pantheon", "pantheon.io"},
	{"ghost.io", "Ghost", "ghost.org"},
	{"hubspot", "HubSpot", "hubspot.com"},
	{"dnsimple.com", "DNSimple", "dnsimple.com"},
	{"pressdns.com", "Pressable", "pressable.com"},
	{"kinsta", "Kinsta", "kinsta.com"},
	{"cdn77", "CDN77", "cdn77.com"},
	{"incapsula", "Imperva", "imperva.com"},
	{"sucuri", "Sucuri", "sucuri.net"},
	{"flywheel", "Flywheel", "getflywheel.com"},
	{"gridscale", "gridscale", "gridscale.io"},
	{"webflow", "Webflow", "webflow.com"},
	{"framer.app", "Framer", "framer.com"},
	{"carrd.co", "Carrd", "carrd.co"},
	{"super.so", "Super", "super.so"},
	{"typedream.app", "Typedream", "typedream.com"},
	{"cargo.site", "Cargo", "cargo.site"},
	{"gitlab.io", "GitLab Pages", "gitlab.com"},
	{"bitbucket.io", "Bitbucket", "bitbucket.org"},
	{"codeberg.page", "Codeberg Pages", "codeberg.org"},
	{"glitch.me", "Glitch", "glitch.com"},
	{"repl.co", "Replit", "replit.com"},
	{"replit.app", "Replit", "replit.com"},
	{"deno.dev", "Deno Deploy", "deno.com"},
	{"supabase.co", "Supabase", "supabase.com"},
	{"firebaseapp.com", "Firebase", "firebase.google.com"},
	{"web.app", "Firebase", "firebase.google.com"},
	{"appspot.com", "Google App Engine", "cloud.google.com"},
}

// asnProviders: ip-api org/AS strings → hosting vendor.
var asnProviders = []provMatch{
	{"cloudflare", "Cloudflare", "cloudflare.com"},
	{"akamai", "Akamai", "akamai.com"},
	{"linode", "Akamai (Linode)", "linode.com"},
	{"amazon", "Amazon AWS", "aws.amazon.com"},
	{"aws", "Amazon AWS", "aws.amazon.com"},
	{"google", "Google Cloud", "cloud.google.com"},
	{"microsoft", "Microsoft Azure", "azure.microsoft.com"},
	{"azure", "Microsoft Azure", "azure.microsoft.com"},
	{"digitalocean", "DigitalOcean", "digitalocean.com"},
	{"hetzner", "Hetzner", "hetzner.com"},
	{"ovh", "OVHcloud", "ovhcloud.com"},
	{"kimsufi", "OVHcloud", "ovhcloud.com"},
	{"soyoustart", "OVHcloud", "ovhcloud.com"},
	{"vultr", "Vultr", "vultr.com"},
	{"choopa", "Vultr", "vultr.com"},
	{"fastly", "Fastly", "fastly.com"},
	{"vercel", "Vercel", "vercel.com"},
	{"netlify", "Netlify", "netlify.com"},
	{"github", "GitHub", "github.com"},
	{"scaleway", "Scaleway", "scaleway.com"},
	{"oracle", "Oracle Cloud", "oracle.com"},
	{"alibaba", "Alibaba Cloud", "alibabacloud.com"},
	{"tencent", "Tencent Cloud", "cloud.tencent.com"},
	{"contabo", "Contabo", "contabo.com"},
	{"ionos", "IONOS", "ionos.com"},
	{"1&1", "IONOS", "ionos.com"},
	{"shopify", "Shopify", "shopify.com"},
	{"netcup", "netcup", "netcup.eu"},
	{"upcloud", "UpCloud", "upcloud.com"},
	{"kamatera", "Kamatera", "kamatera.com"},
	{"fly.io", "Fly.io", "fly.io"},
	{"render", "Render", "render.com"},
	{"railway", "Railway", "railway.app"},
	{"bunny", "Bunny", "bunny.net"},
	{"hostinger", "Hostinger", "hostinger.com"},
	{"siteground", "SiteGround", "siteground.com"},
	{"godaddy", "GoDaddy", "godaddy.com"},
	{"automattic", "WordPress.com", "wordpress.com"},
	{"squarespace", "Squarespace", "squarespace.com"},
	{"wix", "Wix", "wix.com"},
	{"infomaniak", "Infomaniak", "infomaniak.com"},
	{"dreamhost", "DreamHost", "dreamhost.com"},
	{"liquid web", "Liquid Web", "liquidweb.com"},
	{"liquidweb", "Liquid Web", "liquidweb.com"},
	{"inmotion", "InMotion", "inmotionhosting.com"},
	{"unified layer", "Bluehost", "bluehost.com"},
	{"hostgator", "HostGator", "hostgator.com"},
	{"glesys", "GleSYS", "glesys.se"},
	{"exoscale", "Exoscale", "exoscale.com"},
	{"incapsula", "Imperva", "imperva.com"},
	{"sucuri", "Sucuri", "sucuri.net"},
	{"ddos-guard", "DDoS-Guard", "ddos-guard.net"},
	{"huawei", "Huawei Cloud", "huaweicloud.com"},
	{"leaseweb", "Leaseweb", "leaseweb.com"},
	{"aruba", "Aruba", "aruba.it"},
	{"krystal", "Krystal", "krystal.uk"},
	{"namecheap", "Namecheap", "namecheap.com"},
	{"web-hosting.com", "Namecheap", "namecheap.com"},
	{"zendesk", "Zendesk", "zendesk.com"},
	{"kinsta", "Kinsta", "kinsta.com"},
	{"flywheel", "Flywheel", "getflywheel.com"},
	{"wp engine", "WP Engine", "wpengine.com"},
	{"pantheon", "Pantheon", "pantheon.io"},
	{"meta", "Meta", "meta.com"},
	{"gitlab", "GitLab", "gitlab.com"},
	{"stackpath", "StackPath", "stackpath.com"},
	{"cdn77", "CDN77", "cdn77.com"},
	{"imperva", "Imperva", "imperva.com"},
	{"framer", "Framer", "framer.com"},
	{"webflow", "Webflow", "webflow.com"},
	{"readthedocs", "Read the Docs", "readthedocs.io"},
	{"transip", "TransIP", "transip.nl"},
	{"strato", "STRATO", "strato.de"},
	{"all-inkl", "ALL-INKL.COM", "all-inkl.com"},
	{"one.com", "One.com", "one.com"},
	{"netlify", "Netlify", "netlify.com"},
	{"neocities", "Neocities", "neocities.org"},
}

var registrarProviders = []provMatch{
	{"markmonitor", "MarkMonitor", "markmonitor.com"},
	{"namecheap", "Namecheap", "namecheap.com"},
	{"cloudflare", "Cloudflare", "cloudflare.com"},
	{"godaddy", "GoDaddy", "godaddy.com"},
	{"gandi", "Gandi", "gandi.net"},
	{"porkbun", "Porkbun", "porkbun.com"},
	{"ovh", "OVHcloud", "ovhcloud.com"},
	{"name.com", "Name.com", "name.com"},
	{"ionos", "IONOS", "ionos.com"},
	{"1&1", "IONOS", "ionos.com"},
	{"1und1", "IONOS", "ionos.com"},
	{"tucows", "Tucows", "tucows.com"},
	{"squarespace", "Squarespace", "squarespace.com"},
	{"google", "Google Domains", "domains.google"},
	{"dynadot", "Dynadot", "dynadot.com"},
	{"namesilo", "NameSilo", "namesilo.com"},
	{"epik", "Epik", "epik.com"},
	{"hexonet", "HEXONET", "hexonet.net"},
	{"1api", "HEXONET", "hexonet.net"},
	{"openprovider", "Openprovider", "openprovider.com"},
	{"internetx", "InterNetX", "internetx.com"},
	{"nic.cz", "CZ.NIC", "nic.cz"},
	{"hostinger", "Hostinger", "hostinger.com"},
	{"com laude", "Com Laude", "comlaude.com"},
	{"csc", "CSC", "cscdbs.com"},
	{"safenames", "Safenames", "safenames.net"},
	{"xinnet", "Xin Net", "xinnet.com"},
	{"alibaba", "Alibaba Cloud", "alibabacloud.com"},
	{"dnspod", "DNSPod", "dnspod.com"},
	{"tencent", "Tencent Cloud", "cloud.tencent.com"},
	{"gabia", "Gabia", "gabia.com"},
	{"reg.ru", "REG.RU", "reg.ru"},
	{"nic.ru", "RU-CENTER", "nic.ru"},
	{"nordname", "NordName", "nordname.com"},
	{"internet.bs", "Internet.bs", "internetbs.net"},
	{"key-systems", "Key-Systems", "key-systems.net"},
	{"united domains", "United Domains", "uniteddomains.com"},
	{"united-domains", "United Domains", "uniteddomains.com"},
	{"scaleway", "Scaleway", "scaleway.com"},
	{"hetzner", "Hetzner", "hetzner.com"},
	{"infomaniak", "Infomaniak", "infomaniak.com"},
	{"one.com", "One.com", "one.com"},
	{"netim", "Netim", "netim.com"},
	{"inwx", "INWX", "inwx.com"},
	{"internetworx", "INWX", "inwx.com"},
	{"checkdomain", "checkdomain", "checkdomain.de"},
	{"domaindiscount24", "domaindiscount24", "domaindiscount24.com"},
	{"easyname", "easyname", "easyname.com"},
	{"world4you", "World4You", "world4you.com"},
	{"all-inkl", "ALL-INKL.COM", "all-inkl.com"},
	{"transip", "TransIP", "transip.nl"},
	{"strato", "STRATO", "strato.de"},
	{"hover", "Hover", "hover.com"},
	{"enom", "eNom", "enom.com"},
	{"namebright", "NameBright", "namebright.com"},
	{"sav.com", "Sav.com", "sav.com"},
	{"turncommerce", "DropCatch", "dropcatch.com"},
	{"eurodns", "EuroDNS", "eurodns.com"},
	{"gransy", "Gransy", "regtons.com"},
	{"101domain", "101domain", "101domain.com"},
	{"iwantmyname", "iwantmyname", "iwantmyname.com"},
	{"rebel", "Rebel", "rebel.ca"},
	{"blacknight", "Blacknight", "blacknight.com"},
	{"gname", "GNAME", "gname.com"},
	{"west.cn", "West.cn", "west.cn"},
	{"west263", "West.cn", "west.cn"},
	{"imena.ua", "Imena.ua", "imena.ua"},
	{"ukraine.com.ua", "Hosting Ukraine", "ukraine.com.ua"},
	{"beget", "Beget", "beget.com"},
	{"selectel", "Selectel", "selectel.ru"},
	{"timeweb", "Timeweb", "timeweb.ru"},
	{"papaki", "Papaki", "papaki.com"},
	{"dotname", "Dotname", "dotname.co.kr"},
	{"gabia", "Gabia", "gabia.com"},
	{"registar-se", "Namecheap", "namecheap.com"},
	{"registrar-servers", "Namecheap", "namecheap.com"},
	{"namebay", "Namebay", "namebay.com"},
	{"domainpeople", "DomainPeople", "domainpeople.com"},
	{"networksolutions", "Network Solutions", "networksolutions.com"},
	{"network solutions", "Network Solutions", "networksolutions.com"},
	{"register.com", "Register.com", "register.com"},
	{"webcentral", "Webcentral", "webcentral.com.au"},
	{"melbourne it", "Melbourne IT", "melbourneit.com.au"},
	{"corporatedomains", "CSC", "cscdbs.com"},
	{"amazon registrar", "Amazon Registrar", "aws.amazon.com"},
	{"nominet", "Nominet", "nominet.uk"},
	{"123-reg", "123 Reg", "123-reg.co.uk"},
	{"domain.com", "Domain.com", "domain.com"},
	{"dotster", "Dotster", "dotster.com"},
	{"joker", "Joker.com", "joker.com"},
	{"variomedia", "Variomedia", "variomedia.de"},
	{"goneo", "goneo", "goneo.de"},
	{"manitu", "manitu", "manitu.de"},
	{"hosteurope", "Host Europe", "hosteurope.de"},
	{"keyweb", "Keyweb", "keyweb.de"},
	{"wedos", "WEDOS", "wedos.com"},
	{"active24", "Active 24", "active24.com"},
	{"forpsi", "FORPSI", "forpsi.com"},
	{"ignum", "IGNUM", "ignum.cz"},
	{"oneprovider", "OneProvider", "oneprovider.com"},
	{"reg.cz", "REG.CZ", "reg.cz"},
	{"subreg", "Websupport", "websupport.sk"},
	{"websupport", "Websupport", "websupport.sk"},
	{"loopia", "Loopia", "loopia.se"},
	{"binero", "Binero", "binero.se"},
	{"one.se", "One.com", "one.com"},
	{"simply.com", "Simply.com", "simply.com"},
	{"domeneshop", "Domeneshop", "domeneshop.no"},
	{"syse", "Syse", "syse.no"},
	{"registrar.eu", "Openprovider", "openprovider.com"},
	{"mondo", "Mondo International", "mondo.com"},
	{"dansk.net", "Dansk.net", "dansk.net"},
	{"internet invest", "Imena.ua", "imena.ua"},
	{"netearth", "NetEarth One", "netearthone.com"},
	{"publicdomainregistry", "PublicDomainRegistry", "publicdomainregistry.com"},
	{"pdr ltd", "PublicDomainRegistry", "publicdomainregistry.com"},
	{"webnames", "Webnames", "webnames.ca"},
	{"rebel.ca", "Rebel", "rebel.ca"},
	{"easydns", "easyDNS", "easydns.com"},
	{"namespro", "Namespro", "namespro.ca"},
	{"sibername", "Sibername", "sibername.com"},
	{"netfirms", "Netfirms", "netfirms.com"},
	{"names.co.uk", "names.co.uk", "names.co.uk"},
	{"namesco", "names.co.uk", "names.co.uk"},
	{"fasthosts", "Fasthosts", "fasthosts.co.uk"},
	{"heart internet", "Heart Internet", "heartinternet.uk"},
	{"dreamhost", "DreamHost", "dreamhost.com"},
	{"pair domains", "pair Domains", "pairdomains.com"},
	{"domainthenet", "Domain The Net", "domainthenet.com"},
	{"domainregistry", "DomainRegistry", "domainregistry.com"},
	{"moniker", "Moniker", "moniker.com"},
	{"fabulous", "Fabulous", "fabulous.com"},
	{"above.com", "Above.com", "above.com"},
	{"united-internet", "United Internet", "united-internet.de"},
	{"united internet", "United Internet", "united-internet.de"},
	{"arsys", "Arsys", "arsys.es"},
	{"piensa solutions", "Piensa Solutions", "piensasolutions.com"},
	{"dinahosting", "Dinahosting", "dinahosting.com"},
	{"cdmon", "CDmon", "cdmon.com"},
	{"nominalia", "Nominalia", "nominalia.com"},
	{"register.it", "Register.it", "register.it"},
	{"ovh sas", "OVHcloud", "ovhcloud.com"},
	{"gandi sas", "Gandi", "gandi.net"},
	{"blue razor", "GoDaddy", "godaddy.com"},
	{"wild west domains", "GoDaddy", "godaddy.com"},
	{"starfield", "GoDaddy", "godaddy.com"},
	{"nameking", "Namecheap", "namecheap.com"},
	{"enom", "eNom", "enom.com"},
	{"realtime register", "Realtime Register", "realtimeregister.com"},
	{"rrpproxy", "RRPproxy", "rrpproxy.net"},
	{"key-systems gmbh", "Key-Systems", "key-systems.net"},
	{"internet domains", "Internet Domains", "internetdomains.ch"},
	{"corehub", "CORE", "corehub.net"},
	{"don dominio", "DonDominio", "dondominio.com"},
	{"arsys internet", "Arsys", "arsys.es"},
	{"key-systems", "Key-Systems", "key-systems.net"},
	{"centralnic", "CentralNic", "centralnic.com"},
	{"internet.bs corp", "Internet.bs", "internetbs.net"},
	{"gkg.net", "GKG.NET", "gkg.net"},
	{"namecheap inc", "Namecheap", "namecheap.com"},
	{"namecheap, inc", "Namecheap", "namecheap.com"},
	{"google llc", "Google Domains", "domains.google"},
	{"squarespace domains", "Squarespace", "squarespace.com"},
	{"turncommerce, inc.", "DropCatch", "dropcatch.com"},
	{"amazon", "Amazon Registrar", "aws.amazon.com"},
	{"name.com, inc", "Name.com", "name.com"},
	{"godaddy.com, llc", "GoDaddy", "godaddy.com"},
	{"tucows domains", "Tucows", "tucows.com"},
	{"porkbun llc", "Porkbun", "porkbun.com"},
	{"cloudflare, inc", "Cloudflare", "cloudflare.com"},
}

// caProviders: issuer CN/org → certificate authority vendor.
var caProviders = []provMatch{
	{"let's encrypt", "Let's Encrypt", "letsencrypt.org"},
	{"letsencrypt", "Let's Encrypt", "letsencrypt.org"},
	{"digicert", "DigiCert", "digicert.com"},
	{"sectigo", "Sectigo", "sectigo.com"},
	{"comodoca", "Sectigo", "sectigo.com"},
	{"comodo", "Sectigo", "sectigo.com"},
	{"globalsign", "GlobalSign", "globalsign.com"},
	{"godaddy", "GoDaddy", "godaddy.com"},
	{"starfield", "GoDaddy", "godaddy.com"},
	{"entrust", "Entrust", "entrust.com"},
	{"google trust", "Google Trust Services", "pki.goog"},
	{"pki.goog", "Google Trust Services", "pki.goog"},
	{"amazon", "Amazon Trust Services", "amazontrust.com"},
	{"zerossl", "ZeroSSL", "zerossl.com"},
	{"buypass", "Buypass", "buypass.com"},
	{"ssl.com", "SSL.com", "ssl.com"},
	{"actalis", "Actalis", "actalis.com"},
	{"certum", "Certum", "certum.pl"},
	{"harica", "HARICA", "harica.gr"},
	{"swisssign", "SwissSign", "swisssign.com"},
	{"cloudflare", "Cloudflare", "cloudflare.com"},
	{"microsoft", "Microsoft", "microsoft.com"},
	{"geotrust", "DigiCert", "digicert.com"},
	{"thawte", "DigiCert", "digicert.com"},
	{"rapidssl", "DigiCert", "digicert.com"},
	{"usertrust", "Sectigo", "sectigo.com"},
	{"certainly", "Certainly", "certainly.io"},
	{"isrg", "Let's Encrypt", "letsencrypt.org"},
	{"e1", "Let's Encrypt", "letsencrypt.org"},
	{"r3", "Let's Encrypt", "letsencrypt.org"},
	{"r10", "Let's Encrypt", "letsencrypt.org"},
	{"r11", "Let's Encrypt", "letsencrypt.org"},
	{"wotrus", "WoTrus", "wotrus.com"},
	{"cameo", "Cameo", "cameo.com"},
}

// detectHostProvider picks the hosting vendor: CNAME target first (most
// precise), then response headers, then the ASN/org of the primary IPv4.
func detectHostProvider(r *Result) *Provider {
	if r.CNAME != "" {
		if p := matchProvider(r.CNAME, hostTargets); p != nil {
			return p
		}
	}
	h := r.Headers
	server := strings.ToLower(h["server"] + " " + h["x-powered-by"] + " " + h["via"])
	switch {
	case h["cf-ray"] != "" || strings.Contains(server, "cloudflare"):
		return prov("Cloudflare", "cloudflare.com")
	case h["x-vercel-id"] != "" || strings.Contains(server, "vercel"):
		return prov("Vercel", "vercel.com")
	case h["x-nf-request-id"] != "" || strings.Contains(server, "netlify"):
		return prov("Netlify", "netlify.com")
	case h["x-amz-cf-id"] != "" || strings.Contains(server, "cloudfront"):
		return prov("AWS CloudFront", "aws.amazon.com")
	case h["x-github-request-id"] != "" || strings.Contains(server, "github"):
		return prov("GitHub Pages", "github.com")
	case strings.Contains(server, "fly.io") || h["fly-request-id"] != "":
		return prov("Fly.io", "fly.io")
	case strings.Contains(server, "render"):
		return prov("Render", "render.com")
	case strings.Contains(server, "shopify"):
		return prov("Shopify", "shopify.com")
	case strings.Contains(server, "squarespace"):
		return prov("Squarespace", "squarespace.com")
	case strings.Contains(server, "wix"):
		return prov("Wix", "wix.com")
	}
	webServers := []provMatch{
		{"awselb", "AWS ELB", "aws.amazon.com"},
		{"gws", "Google", "google.com"},
		{"kestrel", "Kestrel/.NET", "microsoft.com"},
		{"cowboy", "Cowboy/Erlang", "heroku.com"},
	}
	if p := matchProvider(server, webServers); p != nil {
		return p
	}
	if p := matchProvider(r.HostOrg+" "+r.HostAS+" "+r.HostISP, asnProviders); p != nil {
		return p
	}
	if p := matchProvider(server, []provMatch{
		{"nginx", "nginx", "nginx.org"},
		{"apache", "Apache", "apache.org"},
		{"caddy", "Caddy", "caddyserver.com"},
		{"litespeed", "LiteSpeed", "litespeedtech.com"},
		{"openresty", "OpenResty", "openresty.org"},
		{"envoy", "Envoy", "envoyproxy.io"},
	}); p != nil {
		return p
	}
	return nil
}

// txtProviders: SPF includes and domain-verification strings → vendor.
var txtProviders = []provMatch{
	{"v=spf1 include:_spf.google.com", "Google Workspace", "google.com"},
	{"include:spf.protonmail.ch", "Proton Mail", "proton.me"},
	{"include:mail.protonmail.ch", "Proton Mail", "proton.me"},
	{"include:_spf.mx.cloudflare.net", "Cloudflare", "cloudflare.com"},
	{"include:servers.mcsv.net", "Mailchimp", "mailchimp.com"},
	{"include:amazonses.com", "Amazon SES", "aws.amazon.com"},
	{"include:mailgun.org", "Mailgun", "mailgun.com"},
	{"include:sendgrid.net", "SendGrid", "sendgrid.com"},
	{"include:spf.protection.outlook.com", "Microsoft 365", "microsoft.com"},
	{"include:spf.purelymail.com", "Purelymail", "purelymail.com"},
	{"_spf.purelymail.com", "Purelymail", "purelymail.com"},
	{"purelymail_ownership_proof", "Purelymail", "purelymail.com"},
	{"openai-domain-verification", "OpenAI", "openai.com"},
	{"include:_spf.migadu.com", "Migadu", "migadu.com"},
	{"include:mx.zohomail.com", "Zoho Mail", "zoho.com"},
	{"include:_spf.fastmail.com", "Fastmail", "fastmail.com"},
	{"include:spf.mtasv.net", "Postmark", "postmarkapp.com"},
	{"include:_spf.salesforce.com", "Salesforce", "salesforce.com"},
	{"include:spf.improvmx.com", "ImprovMX", "improvmx.com"},
	{"include:mail.zendesk.com", "Zendesk", "zendesk.com"},
	{"include:spf.mandrillapp.com", "Mailchimp", "mailchimp.com"},
	{"include:mailspike.net", "Mailspike", "mailspike.net"},
	{"include:spf.messagelabs.com", "Broadcom", "broadcom.com"},
	{"include:_spf.mx.yandex", "Yandex", "yandex.com"},
	{"include:spf.tuta.io", "Tuta", "tuta.com"},
	{"google-site-verification", "Google", "google.com"},
	{"ms=", "Microsoft 365", "microsoft.com"},
	{"facebook-domain-verification", "Meta", "meta.com"},
	{"apple-domain-verification", "Apple", "apple.com"},
	{"keybase-site-verification", "Keybase", "keybase.io"},
	{"keybase", "Keybase", "keybase.io"},
	{"protonmail-verification", "Proton", "proton.me"},
	{"have-i-been-pwned-verification", "Have I Been Pwned", "haveibeenpwned.com"},
	{"atlassian-domain-verification", "Atlassian", "atlassian.com"},
	{"docker-verification", "Docker", "docker.com"},
	{"docusign", "DocuSign", "docusign.com"},
	{"yandex-verification", "Yandex", "yandex.com"},
	{"stripe-verification", "Stripe", "stripe.com"},
	{"dropbox-domain-verification", "Dropbox", "dropbox.com"},
	{"brave-ledger-verification", "Brave", "brave.com"},
	{"detectify-verification", "Detectify", "detectify.com"},
	{"adobe-idp-site-verification", "Adobe", "adobe.com"},
	{"webexdomainverification", "Cisco Webex", "webex.com"},
	{"zoho-verification", "Zoho", "zoho.com"},
	{"mailru-verification", "Mail.ru", "mail.ru"},
	{"baidu-site-verification", "Baidu", "baidu.com"},
	{"google-gws-recovery-domain-verification", "Google", "google.com"},
	{"status-page-domain-verification", "Atlassian", "atlassian.com"},
	{"citrix-verification", "Citrix", "citrix.com"},
	{"tiktok-domain-verification", "TikTok", "tiktok.com"},
	{"netlify-verification", "Netlify", "netlify.com"},
	{"vercel-domain-verification", "Vercel", "vercel.com"},
	{"workplace-domain-verification", "Meta", "meta.com"},
	{"v=dmarc1", "DMARC", "dmarc.org"},
	{"v=dkim1", "DKIM", "dkim.org"},
}

func matchTXTProvider(txt string) *Provider {
	return matchProvider(txt, txtProviders)
}
