package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

// diBase mirrors the frontend's dashboard-icons CDN root (ServiceDialog).
const diBase = "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png"

// importExternal ingests a foreign dashboard config — Homepage services.yml,
// Homarr JSON export, or Dashy conf.yml — and replaces the board with it.
// Format is sniffed from the body; settings are left untouched.
func (s *Server) importExternal(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 8<<20))
	if err != nil || len(body) == 0 {
		fail(c, http.StatusBadRequest, "empty or unreadable body")
		return
	}
	sections, err := parseExternal(body)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.replaceBoard(sections, nil, nil); err != nil {
		failImport(c, err)
		return
	}
	board, err := s.board("")
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, board)
}

// parseExternal sniffs the config format: JSON decodes as Homarr; a YAML
// sequence is Homepage's grouped services list; a YAML mapping carrying
// `sections` is a Dashy conf.yml.
func parseExternal(body []byte) ([]Section, error) {
	var j any
	if json.Unmarshal(body, &j) == nil {
		return parseHomarr(j)
	}
	var y any
	if err := yaml.Unmarshal(body, &y); err != nil {
		return nil, errors.New("unreadable file — expected JSON or YAML")
	}
	switch root := y.(type) {
	case []any:
		return parseHomepage(root)
	case map[string]any:
		if _, ok := root["sections"]; ok {
			return parseDashy(body)
		}
	}
	return nil, errors.New("unrecognized config — expected a Homepage services.yml, Homarr JSON export, or Dashy conf.yml")
}

// ---------- Homepage ----------

// parseHomepage walks a services.yml: a sequence of single-key maps
// (group name -> sequence of single-key service maps). Bare single-key
// entries at the top level land in an "Imported" section.
func parseHomepage(groups []any) ([]Section, error) {
	var secs []Section
	loose := Section{Name: "Imported"}
	for _, g := range groups {
		gm, ok := g.(map[string]any)
		if !ok {
			continue
		}
		for name, v := range gm {
			switch val := v.(type) {
			case []any:
				sec := Section{Name: name, Items: homepageItems(val)}
				if len(sec.Items) > 0 {
					secs = append(secs, sec)
				}
			case map[string]any:
				if it, ok := homepageService(name, val); ok {
					loose.Items = append(loose.Items, it)
				}
			}
		}
	}
	if len(loose.Items) > 0 {
		secs = append([]Section{loose}, secs...)
	}
	if len(secs) == 0 {
		return nil, errors.New("homepage: no importable services found")
	}
	return assignIDs(secs), nil
}

func homepageItems(list []any) []Item {
	var items []Item
	for _, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for name, fields := range m {
			fm, _ := fields.(map[string]any)
			if it, ok := homepageService(name, fm); ok {
				items = append(items, it)
			}
		}
	}
	return items
}

// homepageService maps one service entry. Entries without a reachable URL
// (pure widget config, bare ping targets) are skipped rather than failing
// the whole import.
func homepageService(name string, fields map[string]any) (Item, bool) {
	u := strOf(fields, "href", "siteMonitor", "ping")
	if strings.TrimSpace(name) == "" || !validURL(u) {
		return Item{}, false
	}
	return importedItem(name, resolveIcon(strOf(fields, "icon"), u), u), true
}

// ---------- Homarr ----------

// parseHomarr tolerates Homarr's shifting export shapes: `categories[].apps`,
// a flat `apps` array grouped by category id, or board `sections[].items`.
func parseHomarr(root any) ([]Section, error) {
	m, ok := root.(map[string]any)
	if !ok {
		return nil, errors.New("homarr: expected a JSON object")
	}
	var secs []Section
	seen := map[string]bool{}
	catNames := map[string]string{}

	for _, c := range arrOf(m["categories"]) {
		cm, _ := c.(map[string]any)
		if name := strOf(cm, "name"); name != "" {
			catNames[strOf(cm, "id")] = name
			if s := sectionFromApps(name, arrOf(cm["apps"]), seen); len(s.Items) > 0 {
				secs = append(secs, s)
			}
		}
	}

	if apps := arrOf(m["apps"]); len(apps) > 0 {
		var order []string
		byCat := map[string][]any{}
		for _, a := range apps {
			am, _ := a.(map[string]any)
			cat := strOf(am, "categoryId", "category")
			if _, ok := byCat[cat]; !ok {
				order = append(order, cat)
			}
			byCat[cat] = append(byCat[cat], a)
		}
		for _, cat := range order {
			name := catNames[cat]
			if name == "" {
				name = "Imported"
			}
			if s := sectionFromApps(name, byCat[cat], seen); len(s.Items) > 0 {
				secs = append(secs, s)
			}
		}
	}

	for _, raw := range arrOf(m["sections"]) {
		sm, _ := raw.(map[string]any)
		if s := sectionFromApps(strOf(sm, "name"), arrOf(sm["items"]), seen); len(s.Items) > 0 {
			secs = append(secs, s)
		}
	}

	if len(secs) == 0 {
		return nil, errors.New("homarr: no apps found")
	}
	return assignIDs(secs), nil
}

func sectionFromApps(name string, apps []any, seen map[string]bool) Section {
	sec := Section{Name: orDefault(name, "Imported")}
	for _, a := range apps {
		am, _ := a.(map[string]any)
		it, ok := homarrItem(am)
		if !ok || seen[it.Name+"|"+it.URLs[0].URL] {
			continue
		}
		seen[it.Name+"|"+it.URLs[0].URL] = true
		sec.Items = append(sec.Items, it)
	}
	return sec
}

func homarrItem(m map[string]any) (Item, bool) {
	name := strOf(m, "name", "title", "label")
	u := strOf(m, "url", "href", "appUrl", "externalUrl", "internalUrl")
	if name == "" || !validURL(u) {
		return Item{}, false
	}
	return importedItem(name, resolveIcon(strOf(m, "icon", "iconUrl"), u), u), true
}

// ---------- Dashy ----------

type dashyConf struct {
	Sections []struct {
		Name  string `yaml:"name"`
		Items []struct {
			Title string `yaml:"title"`
			URL   string `yaml:"url"`
			Icon  string `yaml:"icon"`
		} `yaml:"items"`
	} `yaml:"sections"`
}

func parseDashy(body []byte) ([]Section, error) {
	var conf dashyConf
	if err := yaml.Unmarshal(body, &conf); err != nil {
		return nil, fmt.Errorf("dashy: malformed conf.yml")
	}
	var secs []Section
	for _, ds := range conf.Sections {
		sec := Section{Name: orDefault(ds.Name, "Imported")}
		for _, di := range ds.Items {
			if di.Title == "" || !validURL(di.URL) {
				continue
			}
			sec.Items = append(sec.Items, importedItem(di.Title, resolveIcon(di.Icon, di.URL), di.URL))
		}
		secs = append(secs, sec)
	}
	if len(secs) == 0 {
		return nil, errors.New("dashy: no sections found")
	}
	return assignIDs(secs), nil
}

// ---------- shared ----------

// assignIDs stamps fresh ids and 1024-spaced positions on an imported tree.
func assignIDs(secs []Section) []Section {
	for si := range secs {
		secs[si].ID = newID("s")
		secs[si].Position = float64(si+1) * 1024
		for ii := range secs[si].Items {
			it := &secs[si].Items[ii]
			it.ID = newID("i")
			it.SectionID = secs[si].ID
			it.Kind = orDefault(it.Kind, "service")
			it.Position = float64(ii+1) * 1024
			for ui := range it.URLs {
				it.URLs[ui].ID = newID("u")
				it.URLs[ui].Position = float64(ui+1) * 1024
			}
		}
	}
	return secs
}

func importedItem(name, icon, u string) Item {
	return Item{Kind: "service", Name: name, Icon: icon, URLs: []URL{{URL: u}}}
}

// resolveIcon maps a foreign icon reference to a usable URL: passthrough for
// absolute URLs, favicon derivation for Dashy's `favicon`, the dashboard-icons
// CDN for prefixed or bare slugs, and "" (letter tile) for local file paths
// that only existed on the old dashboard.
func resolveIcon(icon, itemURL string) string {
	icon = strings.TrimSpace(icon)
	switch {
	case icon == "":
		return ""
	case strings.HasPrefix(icon, "http://"), strings.HasPrefix(icon, "https://"):
		return icon
	case icon == "favicon", icon == "generalfavicon":
		return faviconOf(itemURL)
	case strings.ContainsAny(icon, "/.\\"):
		return ""
	}
	lower := strings.ToLower(icon)
	for _, p := range []string{"si-", "sh-", "hl-", "di-", "mdi-"} {
		if rest, ok := strings.CutPrefix(lower, p); ok {
			if rest == "" {
				return ""
			}
			return diBase + "/" + rest + ".png"
		}
	}
	return diBase + "/" + lower + ".png"
}

func faviconOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/favicon.ico"
}

func arrOf(v any) []any {
	a, _ := v.([]any)
	return a
}

func strOf(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}
