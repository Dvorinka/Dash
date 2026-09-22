package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdvorak/dash/internal/domain"
	"github.com/tdvorak/dash/internal/widget"
	"go.uber.org/zap"
)

// ---------- model ----------

const domainCols = `id, name, tld, active, auto_renew, alert_days_before, tags, notes,
	expiry_date, creation_date, updated_date, registrar_name, registrar_id,
	registrar_url, registry_domain_id, dnssec, statuses, privacy_enabled,
	transfer_lock, registrant_name, registrant_org, registrant_country,
	abuse_email, name_servers, mx_records, txt_records, cname, ipv4, ipv6,
	ssl_issuer, ssl_valid_from, ssl_valid_to, ssl_subject, ssl_fingerprint,
	ssl_key_size, ssl_sig_algo, ssl_alt_names, host_country, host_country_code,
	host_region, host_city, host_isp, host_org, host_as, host_lat, host_lon,
	dns_provider, email_provider, hosting_provider, ca_provider, headers,
	records, providers,
	favicon_url, lookup_error, interval_h, last_checked, position, alerts, created_at`

// domainRow mirrors the domains table; JSON text columns decode on scan.
type domainRow struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	TLD              string   `json:"tld"`
	Active           bool     `json:"active"`
	AutoRenew        bool     `json:"autoRenew"`
	AlertDaysBefore  int      `json:"alertDaysBefore"`
	Tags             []string `json:"tags"`
	Notes            string   `json:"notes"`
	ExpiryDate       *string  `json:"expiryDate"`
	CreationDate     *string  `json:"creationDate"`
	UpdatedDate      *string  `json:"updatedDate"`
	RegistrarName    string   `json:"registrarName"`
	RegistrarID      string   `json:"registrarId"`
	RegistrarURL     string   `json:"registrarUrl"`
	RegistryDomainID string   `json:"registryDomainId"`
	DNSSEC           string   `json:"dnssec"`
	Statuses         []string `json:"statuses"`
	PrivacyEnabled   bool     `json:"privacyEnabled"`
	TransferLock     bool     `json:"transferLock"`
	RegistrantName   string   `json:"registrantName"`
	RegistrantOrg    string   `json:"registrantOrg"`
	RegistrantCountry string  `json:"registrantCountry"`
	AbuseEmail       string   `json:"abuseEmail"`
	NameServers      []string `json:"nameServers"`
	MXRecords        []string `json:"mxRecords"`
	TXTRecords       []string `json:"txtRecords"`
	CNAME            string   `json:"cname"`
	IPv4             []string `json:"ipv4"`
	IPv6             []string `json:"ipv6"`
	SSLIssuer        string   `json:"sslIssuer"`
	SSLValidFrom     *string  `json:"sslValidFrom"`
	SSLValidTo       *string  `json:"sslValidTo"`
	SSLSubject       string   `json:"sslSubject"`
	SSLFingerprint   string   `json:"sslFingerprint"`
	SSLKeySize       int      `json:"sslKeySize"`
	SSLSigAlgo       string   `json:"sslSigAlgo"`
	SSLAltNames      []string `json:"sslAltNames"`
	HostCountry      string   `json:"hostCountry"`
	HostCountryCode  string   `json:"hostCountryCode"`
	HostRegion       string   `json:"hostRegion"`
	HostCity         string   `json:"hostCity"`
	HostISP          string   `json:"hostIsp"`
	HostOrg          string   `json:"hostOrg"`
	HostAS           string   `json:"hostAs"`
	HostLat          float64  `json:"hostLat"`
	HostLon          float64  `json:"hostLon"`
	DNSProvider      string   `json:"dnsProvider"`
	EmailProvider    string   `json:"emailProvider"`
	HostingProvider  string   `json:"hostingProvider"`
	CAProvider       string   `json:"caProvider"`
	Headers          map[string]string `json:"headers"`
	Records          []domain.DNSRecord `json:"records"`
	Providers        domain.Providers   `json:"providers"`
	FaviconURL       string   `json:"faviconUrl"`
	LookupError      string   `json:"lookupError"`
	IntervalH        int      `json:"intervalH"`
	LastChecked      *string  `json:"lastChecked"`
	Position         float64  `json:"position"`
	Alerts           json.RawMessage `json:"alerts"`
	CreatedAt        string   `json:"createdAt"`
}

func scanDomain(row interface{ Scan(...any) error }) (*domainRow, error) {
	var d domainRow
	var tags, statuses, ns, mx, txt, v4, v6, altNames, headers, records, providers, alerts string
	var active, autoRenew, privacy, lock int
	err := row.Scan(&d.ID, &d.Name, &d.TLD, &active, &autoRenew, &d.AlertDaysBefore,
		&tags, &d.Notes, &d.ExpiryDate, &d.CreationDate, &d.UpdatedDate,
		&d.RegistrarName, &d.RegistrarID, &d.RegistrarURL, &d.RegistryDomainID,
		&d.DNSSEC, &statuses, &privacy, &lock, &d.RegistrantName, &d.RegistrantOrg,
		&d.RegistrantCountry, &d.AbuseEmail, &ns, &mx, &txt, &d.CNAME, &v4, &v6,
		&d.SSLIssuer, &d.SSLValidFrom, &d.SSLValidTo, &d.SSLSubject, &d.SSLFingerprint,
		&d.SSLKeySize, &d.SSLSigAlgo, &altNames, &d.HostCountry, &d.HostCountryCode,
		&d.HostRegion, &d.HostCity, &d.HostISP, &d.HostOrg, &d.HostAS, &d.HostLat,
		&d.HostLon, &d.DNSProvider, &d.EmailProvider, &d.HostingProvider, &d.CAProvider,
		&headers, &records, &providers, &d.FaviconURL, &d.LookupError, &d.IntervalH, &d.LastChecked,
		&d.Position, &alerts, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	d.Alerts = json.RawMessage(alerts)
	d.Active = active != 0
	d.AutoRenew = autoRenew != 0
	d.PrivacyEnabled = privacy != 0
	d.TransferLock = lock != 0
	d.Tags = decodeList(tags)
	d.Statuses = decodeList(statuses)
	d.NameServers = decodeList(ns)
	d.MXRecords = decodeList(mx)
	d.TXTRecords = decodeList(txt)
	d.IPv4 = decodeList(v4)
	d.IPv6 = decodeList(v6)
	d.SSLAltNames = decodeList(altNames)
	d.Headers = map[string]string{}
	_ = json.Unmarshal([]byte(headers), &d.Headers)
	d.Records = []domain.DNSRecord{}
	_ = json.Unmarshal([]byte(records), &d.Records)
	_ = json.Unmarshal([]byte(providers), &d.Providers)
	return &d, nil
}

func decodeList(s string) []string {
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		out = []string{}
	}
	return out
}

func (s *Server) loadDomain(id string) (*domainRow, error) {
	return scanDomain(s.db.QueryRow(`SELECT `+domainCols+` FROM domains WHERE id = ?`, id))
}

// domainView adds computed countdowns the cards/detail page render directly.
type domainView struct {
	*domainRow
	DaysUntilExpiry    *int `json:"daysUntilExpiry"`
	SSLDaysUntilExpiry *int `json:"sslDaysUntilExpiry"`
	Expiring           bool `json:"expiring"`
	SSLExpiring        bool `json:"sslExpiring"`
}

func daysUntil(iso *string) *int {
	if iso == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *iso)
	if err != nil {
		if t, err = time.Parse("2006-01-02", *iso); err != nil {
			return nil
		}
	}
	d := int(time.Until(t).Hours() / 24)
	return &d
}

func viewOf(d *domainRow) domainView {
	v := domainView{domainRow: d}
	v.DaysUntilExpiry = daysUntil(d.ExpiryDate)
	v.SSLDaysUntilExpiry = daysUntil(d.SSLValidTo)
	if v.DaysUntilExpiry != nil {
		v.Expiring = *v.DaysUntilExpiry <= d.AlertDaysBefore
	}
	if v.SSLDaysUntilExpiry != nil {
		v.SSLExpiring = *v.SSLDaysUntilExpiry <= d.AlertDaysBefore
	}
	return v
}

// ---------- handlers ----------

func (s *Server) listDomainsH(c *gin.Context) {
	rows, err := s.db.Query(`SELECT ` + domainCols + ` FROM domains ORDER BY position, name`)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []domainView{}
	for rows.Next() {
		if d, err := scanDomain(rows); err == nil {
			out = append(out, viewOf(d))
		}
	}
	c.JSON(http.StatusOK, out)
}

type domainInput struct {
	Name            *string  `json:"name"`
	Active          *bool    `json:"active"`
	AutoRenew       *bool    `json:"autoRenew"`
	AlertDaysBefore *int     `json:"alertDaysBefore"`
	IntervalH       *int     `json:"intervalH"`
	Tags            []string `json:"tags"`
	Notes           *string  `json:"notes"`
	Alerts          json.RawMessage `json:"alerts"`
}

func (s *Server) createDomain(c *gin.Context) {
	var in domainInput
	if err := c.ShouldBindJSON(&in); err != nil || in.Name == nil {
		fail(c, http.StatusBadRequest, "name required")
		return
	}
	name := domain.Clean(*in.Name)
	if !domain.Valid(name) {
		fail(c, http.StatusBadRequest, "invalid domain name")
		return
	}
	id := newID("d")
	tags, _ := json.Marshal(in.Tags)
	if in.Tags == nil {
		tags = []byte("[]")
	}
	if in.Alerts != nil {
		var rules alertRules
		if err := json.Unmarshal(in.Alerts, &rules); err != nil {
			fail(c, http.StatusBadRequest, "alerts must be an object")
			return
		}
	}
	_, err := s.db.Exec(`INSERT INTO domains
		(id, name, tld, active, auto_renew, alert_days_before, tags, notes, interval_h, position)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		id, name, tldFor(name), boolOr2(in.Active, true), boolOr(in.AutoRenew),
		intOr2(in.AlertDaysBefore, 30), string(tags), strOr(in.Notes),
		intOr2(in.IntervalH, 24), float64(time.Now().UnixNano())/1e9)
	if err != nil {
		fail(c, http.StatusConflict, "domain already tracked or insert failed")
		return
	}
	if in.Alerts != nil {
		_, _ = s.db.Exec(`UPDATE domains SET alerts = ? WHERE id = ?`, string(in.Alerts), id)
	}
	// First lookup runs inline so the response carries real data.
	s.refreshDomain(id)
	d, err := s.loadDomain(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, viewOf(d))
}

func tldFor(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return ""
}

func (s *Server) getDomain(c *gin.Context) {
	d, err := s.loadDomain(c.Param("id"))
	if notFound(err) {
		fail(c, http.StatusNotFound, "domain not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, viewOf(d))
}

func (s *Server) patchDomain(c *gin.Context) {
	id := c.Param("id")
	if _, err := s.loadDomain(id); notFound(err) {
		fail(c, http.StatusNotFound, "domain not found")
		return
	}
	var in domainInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Name != nil {
		name := domain.Clean(*in.Name)
		if !domain.Valid(name) {
			fail(c, http.StatusBadRequest, "invalid domain name")
			return
		}
		_, _ = s.db.Exec(`UPDATE domains SET name = ?, tld = ? WHERE id = ?`, name, tldFor(name), id)
	}
	if in.Notes != nil {
		_, _ = s.db.Exec(`UPDATE domains SET notes = ? WHERE id = ?`, *in.Notes, id)
	}
	if in.AlertDaysBefore != nil {
		_, _ = s.db.Exec(`UPDATE domains SET alert_days_before = ? WHERE id = ?`, *in.AlertDaysBefore, id)
	}
	if in.IntervalH != nil {
		_, _ = s.db.Exec(`UPDATE domains SET interval_h = ? WHERE id = ?`, *in.IntervalH, id)
	}
	if in.Active != nil {
		_, _ = s.db.Exec(`UPDATE domains SET active = ? WHERE id = ?`, *in.Active, id)
	}
	if in.AutoRenew != nil {
		_, _ = s.db.Exec(`UPDATE domains SET auto_renew = ? WHERE id = ?`, *in.AutoRenew, id)
	}
	if in.Tags != nil {
		tags, _ := json.Marshal(in.Tags)
		_, _ = s.db.Exec(`UPDATE domains SET tags = ? WHERE id = ?`, string(tags), id)
	}
	if in.Alerts != nil {
		var rules alertRules
		if err := json.Unmarshal(in.Alerts, &rules); err != nil {
			fail(c, http.StatusBadRequest, "alerts must be an object")
			return
		}
		_, _ = s.db.Exec(`UPDATE domains SET alerts = ? WHERE id = ?`, string(in.Alerts), id)
	}
	d, err := s.loadDomain(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, viewOf(d))
}

func (s *Server) deleteDomain(c *gin.Context) {
	res, err := s.db.Exec(`DELETE FROM domains WHERE id = ?`, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, "domain not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// refreshDomain re-runs the full lookup and writes a snapshot row. Expiry
// threshold crossings fire notify events on the edge (only when a state
// newly becomes true, not on every refresh).
func (s *Server) refreshDomain(id string) {
	row, err := s.loadDomain(id)
	if err != nil {
		return
	}
	prev := viewOf(row)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	r := s.lookup(ctx, row.Name)
	s.applyResult(id, r)

	next, err := s.loadDomain(id)
	if err != nil {
		return
	}
	nv := viewOf(next)
	rules := parseAlertRules(next.Alerts)
	if rules.Mute {
		return
	}
	switch {
	case nv.DaysUntilExpiry != nil && *nv.DaysUntilExpiry < 0 && !(prev.DaysUntilExpiry != nil && *prev.DaysUntilExpiry < 0):
		s.notify("domain.expired", next.Name, fmt.Sprintf("Domain %s has expired", next.Name))
	case nv.Expiring && !prev.Expiring:
		s.notify("domain.expiring", next.Name,
			fmt.Sprintf("Domain %s expires in %d days", next.Name, *nv.DaysUntilExpiry))
	}
	// TLS threshold: alerts.certDays overrides alertDaysBefore.
	certDays := next.AlertDaysBefore
	if rules.CertDays > 0 {
		certDays = rules.CertDays
	}
	sslNow := nv.SSLDaysUntilExpiry != nil && *nv.SSLDaysUntilExpiry <= certDays
	sslPrev := prev.SSLDaysUntilExpiry != nil && *prev.SSLDaysUntilExpiry <= certDays
	if sslNow && !sslPrev {
		s.notify("domain.sslExpiring", next.Name,
			fmt.Sprintf("TLS certificate for %s expires in %d days", next.Name, *nv.SSLDaysUntilExpiry))
	}
	// Subdomain discovery rides the refresh schedule, gated to once/day.
	s.maybeSweepSubdomains(next)
}

func (s *Server) refreshDomainH(c *gin.Context) {
	id := c.Param("id")
	if _, err := s.loadDomain(id); notFound(err) {
		fail(c, http.StatusNotFound, "domain not found")
		return
	}
	s.refreshDomain(id)
	d, err := s.loadDomain(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, viewOf(d))
}

// applyResult persists a lookup Result and appends a domain_checks snapshot.
func (s *Server) applyResult(id string, r *domain.Result) {
	enc := func(v []string) string {
		b, _ := json.Marshal(v)
		return string(b)
	}
	var expiry, created, updated, sslFrom, sslTo *string
	fmtT := func(t *time.Time) *string {
		if t == nil {
			return nil
		}
		s := t.UTC().Format(time.RFC3339)
		return &s
	}
	expiry, created, updated = fmtT(r.ExpiryDate), fmtT(r.CreationDate), fmtT(r.UpdatedDate)
	sslFrom, sslTo = fmtT(r.SSLValidFrom), fmtT(r.SSLValidTo)
	hdr, _ := json.Marshal(r.Headers)
	recs, _ := json.Marshal(r.Records)
	provs, _ := json.Marshal(r.Providers)

	_, err := s.db.Exec(`UPDATE domains SET
		expiry_date=?, creation_date=?, updated_date=?, registrar_name=?,
		registrar_id=?, registrar_url=?, registry_domain_id=?, dnssec=?,
		statuses=?, privacy_enabled=?, transfer_lock=?, registrant_name=?,
		registrant_org=?, registrant_country=?, abuse_email=?, name_servers=?,
		mx_records=?, txt_records=?, cname=?, ipv4=?, ipv6=?, ssl_issuer=?,
		ssl_valid_from=?, ssl_valid_to=?, ssl_subject=?, ssl_fingerprint=?,
		ssl_key_size=?, ssl_sig_algo=?, ssl_alt_names=?, host_country=?,
		host_country_code=?, host_region=?, host_city=?, host_isp=?, host_org=?,
		host_as=?, host_lat=?, host_lon=?, dns_provider=?, email_provider=?,
		hosting_provider=?, ca_provider=?, headers=?, records=?, providers=?,
		favicon_url=?,
		lookup_error=?, last_checked=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id=?`,
		expiry, created, updated, r.RegistrarName, r.RegistrarID, r.RegistrarURL,
		r.RegistryDomainID, r.DNSSEC, enc(r.Statuses), r.PrivacyEnabled,
		r.TransferLock, r.RegistrantName, r.RegistrantOrg, r.RegistrantCountry,
		r.AbuseEmail, enc(r.NameServers), enc(r.MXRecords), enc(r.TXTRecords),
		r.CNAME, enc(r.IPv4), enc(r.IPv6), r.SSLIssuer, sslFrom, sslTo,
		r.SSLSubject, r.SSLFingerprint, r.SSLKeySize, r.SSLSigAlgo,
		enc(r.SSLAltNames), r.HostCountry, r.HostCountryCode, r.HostRegion,
		r.HostCity, r.HostISP, r.HostOrg, r.HostAS, r.HostLat, r.HostLon,
		r.DNSProvider, r.EmailProvider, r.HostingProvider, r.CAProvider,
		string(hdr), string(recs), string(provs), r.FaviconURL, truncate(r.Error, 300), id)
	if err != nil {
		s.log.Error("domain apply", zap.Error(err))
		return
	}
	_, _ = s.db.Exec(`INSERT INTO domain_checks (domain_id, expiry_date, ssl_valid_to, ipv4, name_servers)
		VALUES (?,?,?,?,?)`, id, expiry, sslTo, enc(r.IPv4), enc(r.NameServers))
}

// domainChecks returns the snapshot history — powers change timelines later.
func (s *Server) domainChecks(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT id, expiry_date, ssl_valid_to, ipv4, name_servers, checked_at
		FROM domain_checks WHERE domain_id = ? ORDER BY checked_at DESC LIMIT ?`,
		c.Param("id"), limit)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	type chk struct {
		ID          int      `json:"id"`
		ExpiryDate  *string  `json:"expiryDate"`
		SSLValidTo  *string  `json:"sslValidTo"`
		IPv4        []string `json:"ipv4"`
		NameServers []string `json:"nameServers"`
		CheckedAt   string   `json:"checkedAt"`
	}
	out := []chk{}
	for rows.Next() {
		var k chk
		var v4, ns string
		if err := rows.Scan(&k.ID, &k.ExpiryDate, &k.SSLValidTo, &v4, &ns, &k.CheckedAt); err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		k.IPv4, k.NameServers = decodeList(v4), decodeList(ns)
		out = append(out, k)
	}
	c.JSON(http.StatusOK, out)
}

// ---------- scheduler ----------

// sweepDomains refreshes domains whose interval_h has elapsed. Runs inside
// the monitor tick — the SQL gate keeps it near-free when nothing is due.
func (s *Server) sweepDomains() {
	rows, err := s.db.Query(`SELECT ` + domainCols + ` FROM domains
		WHERE active = 1
		AND (last_checked IS NULL OR datetime(last_checked, '+' || interval_h || ' hours') <= datetime('now'))`)
	if err != nil {
		s.log.Error("domain sweep", zap.Error(err))
		return
	}
	var due []domainRow
	for rows.Next() {
		if d, err := scanDomain(rows); err == nil {
			due = append(due, *d)
		}
	}
	rows.Close()
	for i := range due {
		checkSem <- struct{}{}
		go func(id string) {
			defer func() { <-checkSem }()
			s.refreshDomain(id)
		}(due[i].ID)
	}
}

// ---------- widget fetcher ----------

// domainWidget serves board tiles bound to a domain via config.domainId.
type domainWidget struct{ db *sql.DB }

func (w *domainWidget) Meta() widget.Type {
	return widget.Type{
		Type: "domain", Name: "Domain",
		Description: "Expiry countdown for a tracked domain",
		Fields: []widget.Field{
			{Key: "domainId", Label: "Domain", Required: true},
		},
	}
}

func (w *domainWidget) Fetch(_ context.Context, cfg json.RawMessage) (any, error) {
	var c struct {
		DomainID string `json:"domainId"`
	}
	if err := json.Unmarshal(cfg, &c); err != nil || c.DomainID == "" {
		return nil, errors.New("config.domainId required")
	}
	d, err := scanDomain(w.db.QueryRow(`SELECT `+domainCols+` FROM domains WHERE id = ?`, c.DomainID))
	if err != nil {
		return nil, errors.New("domain not found")
	}
	v := viewOf(d)
	return map[string]any{
		"name":            d.Name,
		"faviconUrl":      d.FaviconURL,
		"daysUntilExpiry": v.DaysUntilExpiry,
		"sslDaysUntilExpiry": v.SSLDaysUntilExpiry,
		"expiring":        v.Expiring,
		"registrar":       d.RegistrarName,
	}, nil
}
