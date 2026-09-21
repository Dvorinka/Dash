# Dash — Roadmap

A self-hosted homelab dashboard. CasaOS-grade ease of use, Vercel-grade design.
Single container, UI-managed, no YAML, no auth required.

Stack: **Go + Gin + SQLite** (single static binary, embeds frontend) · **React 18 + Vite + TS strict + Tailwind + shadcn/ui + dnd-kit** · OpenAPI-generated client.

---

## Design system

Multiple board styles are a core feature: same system, same functionality,
different UI structure per style. A **renderer** supplies the layout; board
state, drag-drop, popovers, and dialogs are shared primitives underneath.

| Renderer | Source mockup | Ships |
|---|---|---|
| Bento (default) | `d-bento-mono.html` | v0.1.0 — mixed-size grid, widgets inline |
| Cards | `a-vercel-mono.html` | v0.1.0 — uniform compact grid; validates the seam early |
| Index | `e-editorial.html` | v0.3.0 — serif headers, list rows, stat strip |
| Console | `f-console.html` | v0.3.0 — terminal panels, `[ up ]` tags |

Contract: `renderers/<name>/{BoardView, SectionView, ItemView, WidgetView}`.
Tokens (`--bg`, `--surface`, `--border`, …) shared; dark/light is orthogonal.
User picks renderer in settings; persisted. Renderers never reimplement dnd
or popover logic.

---

## Releases

| Version | Contents | Gate |
|---|---|---|
| `v0.1.0` | Phase 0 + 1 — working MVP, single image | manual smoke test ✅ |
| `v0.2.0` | Phase 2 — widget layer + first integrations | Pi-hole widget live ✅ |
| `v0.3.0` | Phase 3 — four renderers, ⌘K palette, icon suggest | palette + switcher live ✅ |
| `v0.4.0` | Phase M1 — uptime monitors + alerts backbone | monitor lifecycle live |
| `v0.5.0` | Phase M2 — domain intelligence | domain lookup + expiry alerts |
| `v0.6.0` | Phase M3 — system monitoring + agent | agent ingest + charts |
| `v0.7.0` | Phase M4 — incidents, status pages, badges, metrics | public status page live |
| `v1.0.0` | Phase 4 — OSS polish, docs, importers, CI releases | public repo launch |
| `v1.x` | Phase 5 — demand-driven extras only | per-feature |

---

## Phase 0 — Foundation → part of v0.1.0 ✅ done 2026-09-17

Repo hygiene and pipelines before features.

- [x] Monorepo scaffold (`apps/frontend`, `apps/backend`, `packages/api-client`, `infra/`, `data/`)
- [x] MIT `LICENSE`, `README.md`, `.gitignore`, `AGENTS.md`
- [x] `openapi.yaml` stub → generated TS client (`packages/api-client`)
- [x] Backend skeleton: Gin router, `zap` logging, `/api/healthz`, SQLite open + goose migrations runner
- [x] Frontend skeleton: Vite + TS strict + Tailwind + shadcn/ui, theme tokens matching `d-bento-mono.html` CSS vars
- [x] CI (GitHub Actions): `go build ./...`, `go vet ./...`, `go test ./...`, `tsc --noEmit`, `oxlint` + vendored anti-slop rules, `npm run build`
- [x] Multi-stage `Dockerfile` (frontend build → Go build → distroless, `embed.FS` serves UI)
- [x] SPA fallback routing in embedded server (`/` + client-side paths → `index.html`, `/api` isolated)
- [x] `docker-compose.yml` — one service, `./data:/data` volume

**Exit:** `docker compose up` serves a blank board — verified locally (33.6 MB image). CI green on `main` — pending first push.

## Phase 1 — MVP → v0.1.0 ✅ done 2026-09-17

The product's spine: sections, services, drag-drop, multi-URL.

- [x] Schema: `sections`, `items`, `urls`, `settings` (fractional `position REAL` ordering)
- [x] REST API: CRUD for sections/items/urls, `reorder` endpoints, icon upload, `settings`, `export`/`import` JSON
- [x] Renderer contract: shared board store + `BoardView`/`SectionView`/`ItemView`/`WidgetView` seam, dnd + popover as shared primitives
- [x] **Bento** renderer (default): mixed-size grid, hairline section headers, tiles with status chips
- [x] **Cards** renderer: uniform compact grid — second renderer proves the seam while code is small
- [x] dnd-kit: reorder items in a section, move items between sections, reorder sections; persist on drop
- [x] Collapse/expand sections (persisted)
- [x] `UrlPopover`: 1 URL → direct open, 2+ → chooser (`local`/`external`/custom tags)
- [x] `ServiceDialog` — one dialog for add + edit (name, URLs+labels, icon URL or file upload)
- [x] Icon handling: upload to `data/icons/`, remote URL passthrough, letter-tile fallback
- [x] Dark/light toggle (persisted), renderer picker in settings, responsive to ~360px
- [x] Pre-paint theme bootstrap in `index.html` (no wrong-theme flash on load)
- [x] Status ping: server-side HEAD request per service URL, cached ~60s → `up`/`down` chip
- [x] Parallel probing: per-item `sync.WaitGroup` fan-out, 4s client timeout
- [x] URL scheme allowlist (`http`/`https` only — `file://` etc. rejected on create + probe)
- [x] Post-drag click suppression (`lastDropAt` window — drops no longer fire tile activation)
- [x] Mobile polish: status chip collapses to dot-only below 620px
- [x] `api_test.go`: reorder positions, URL scheme rejection, settings round-trip

**Exit:** create sections, add services with icons and dual URLs, drag everything, reload — state persists. `docker run` single image works. Export/import round-trips. — verified: full API + browser smoke (dnd, popover, upload, collapse, themes, both renderers, 360px), `api_test.go` covers reorder/URL-validation/settings.

## Phase 2 — Widgets → v0.2.0 ✅ done 2026-09-17

- [x] Go widget registry: `Widget` interface (`Type()`, `Fetch(cfg)`), `GET /api/widgets/:id/data` with ~30s in-process TTL cache
- [x] `GET /api/widgets/types` → registry metadata drives the add-widget dialog
- [x] `WidgetHost` component: maps `config.type` → React component, polls with backoff
- [x] Widgets as grid items (same `items` table, `kind = 'widget'`, `config JSON`)
- [x] Clock/timezone widget (frontend-only)
- [x] Pi-hole fetcher (blocked %, queries today) — per-day bar series deferred: v5 `api.php` lacks it, needs Pi-hole v6 `/api/history`
- [x] AdGuard Home fetcher (blocked %, queries, avg processing ms)
- [x] Immich fetcher (photo/video count, library size)
- [x] Per-widget config UI (endpoint URL, API key — stored in `config`, never logged)
- [x] `local` type flag: frontend-only widgets (clock) register in `/api/widgets/types` without a fetcher
- [x] Widget tile error/loading states (graceful upstream 502 rendering)
- [x] Widget endpoint tests: type registry, 404 on missing item, 502 on dead upstream, TTL cache hit

**Exit:** Pi-hole widget shows real data; adding a new integration = one Go file + one React component. — verified: registry, cache, 502 path, generic config dialog, clock live in-browser, Pi-hole fetcher against stubbed v5 API.

## Phase 3 — Renderers & ergonomics → v0.3.0 ✅ done 2026-09-17

- [x] **Index** renderer: serif section headers, full-width rows, masthead stat strip
- [x] **Console** renderer: bordered panels/tables, JetBrains Mono, `[ up ]` tags
- [x] Renderer switcher polished (2×2 radio grid with per-renderer descriptions)
- [x] ⌘K command palette: jump to service, add service, toggle theme, switch renderer (⌘K / Ctrl+K / `/`)
- [x] dashboard-icons auto-suggest in icon picker (name → slug → CDN URL chip + live preview)
- [x] Renderer contract `index` prop → row/section numbering without re-deriving position
- [x] Icon URL live preview in service dialog (dead URLs degrade to dashed tile)
- [x] Palette `/` shortcut + header search button for discoverability
- [x] Fonts: Instrument Serif + JetBrains Mono via fontsource (bundled, no CDN dep)
- [x] Fix: global `:focus-visible` moved into `@layer base` (was silently overriding all utility overrides)
- [ ] User services list → prioritized widget backlog (owner-supplied list pending)

**Exit:** all four renderers work off the same board state; palette navigates everything. — verified: both new renderers render/persist/dnd/collapse, palette filters and executes, icon suggest resolves real CDN assets, 360px + light theme, docker image 43.9 MB.

## Merge program — Beszel fork → Dash

Source: the local Beszel fork (`~/Desktop/PROG+HTML/Beszel` — PocketBase hub
with custom monitors/domains/status-pages) plus lissy93/domain-locker as
design inspiration. Upstream references cloned to `../_ref/beszel` and
`../_ref/domain-locker` for consultation only.

**No code moves verbatim.** The fork is PocketBase collections + its own
auth + a WebSocket agent; Dash is Gin + SQLite + goose with no auth. We port
data models, check logic, and lookup pipelines — DB, API, and UI layers are
re-implemented on Dash conventions.

**Deployment reality.** Monitoring needs a persistent process (schedulers),
non-HTTP egress (WHOIS :43, ICMP, SMTP, raw TCP), and a writable DB. None of
that exists on serverless — Vercel cannot host this product, full stop. The
deploy story stays `docker run` + GHCR image + goreleaser binaries; agents
are a second small binary.

**UI model.** The app becomes routed: `/` board (default), `/monitors`,
`/domains`, `/systems`, `/status/:slug` (public, no board chrome). Router:
`wouter` (~2 kB — every route we need, nothing more). Charts: `recharts`.
Board widgets surface monitoring entities through the existing widget
registry — monitor uptime, domain expiry, system gauges — so the board
becomes the summary layer over the monitoring pages.

### Phase M1 — Monitors → v0.4.0

Uptime checking: the backbone everything else hangs off.

- [ ] Schema: `monitors` (name, type, url/hostname/port, method, keyword, json_query, expected_value, interval_s, timeout_s, retries, active, status, tags, notes), `heartbeats` (monitor_id, status, ping_ms, msg, cert_expiry, checked_at — 30-day retention prune)
- [ ] In-process scheduler: due-monitor sweep, bounded worker pool, heartbeat write, status transitions
- [ ] Checkers: `http`/`https`, `tcp`, `ping` (unprivileged ICMP datagram via `x/net/icmp` — distroless has no ping binary), `dns`, `keyword`, `json-query`, `push` (caller-generated ingest URL)
- [ ] API: `/api/monitors` CRUD, `/:id/heartbeats?range=`, pause/resume, check-now
- [ ] Monitors page (status, uptime 24h/30d, ping, interval) + detail (response chart, heartbeat log)
- [ ] Board widget type `monitor` bound to a monitor id
- [ ] ServiceDialog checkbox: create matching http monitor from a service URL
- [ ] The fork's ~30 exotic monitor types were TCP stubs — not ported. Real types only.

### Phase M2 — Domains → v0.5.0

Domain-locker-grade domain intelligence.

- [ ] Schema: `domains` (core columns + `extra` JSON for the long tail), `domain_history` (domain_id, change_type, field, old, new, at)
- [ ] Lookup pipeline ported from fork `hub/domains/whois/lookup.go`, trimmed: RDAP over HTTPS → native WHOIS TCP:43 + parser; SSL chain via `crypto/tls`; DNS (NS/MX/TXT/A/AAAA) via `net.Resolver`; host geo via ip-api; provider detection via fork `detect/providers.go`; favicon
- [ ] Daily scheduler + manual refresh; field diffs recorded to `domain_history`
- [ ] Alerts: `alert_rules` + `notifications` tables; dispatchers — generic webhook (JSON POST), SMTP via `net/smtp`, Discord/Slack presets. Triggers: domain expiry ≤ N days, SSL expiry ≤ N days, monitor down/recovered (M1), system offline (M3)
- [ ] Domains page + detail (expiry countdown, registrar, SSL, DNS, subdomains)
- [ ] Board widget `domain` (expiry countdown)
- [ ] Subdomain discovery: port `subdomain_discovery.go`, opt-in per domain
- [ ] Not ported (v1): whoisxml API, EURid web scraping, SEO/robots parsing, valuation estimates — demand only

### Phase M3 — Systems → v0.6.0

Beszel-style server monitoring, push-based.

- [x] Schema: `systems` (name, token, host, os/arch, last_seen, status, `latest` JSON snapshot), `system_stats` (system_id, ts, payload JSON — one column, no sub-field queries; 7d retention prune)
- [x] Ingest: `POST /api/systems/ingest` with per-system bearer token; offline when silent > max(3×interval, 30s); `system.up`/`system.down` webhook events
- [x] `cmd/dash-agent`: Linux collector — `/proc` (cpu/mem/net/load/uptime), `/sys` hwmon temps, statfs disk, docker.sock container list; POST every `-interval` (default 10s); `-once` debug mode
- [ ] Agent packaging: systemd unit + Dockerfile + release binaries (Phase 4)
- [x] Systems page (status cards, cpu/mem/disk bars, uptime) + detail (recharts: cpu%, mem%, net rx/tx, load; temps + containers tables)
- [x] Board widget `system` (cpu/mem/disk mini bars)
- [x] Not ported: the beszel agent protocol (SSH/WS into PocketBase) — our agent is push-JSON. SMART/ZFS/GPU metrics + per-container cpu/mem deferred to demand.

### Phase M4 — Ops layer → v0.7.0

- [x] Incidents: `incidents` + `incident_updates`; manual CRUD + auto-open on monitor down, auto-resolve on recovery; severity + status flow (open → ack → resolved → closed)
- [x] Maintenance windows: `maintenance_windows` (monitor_ids empty = all); suppress alerts + auto-incidents, status pages show `maintenance` state while active
- [x] Status pages: `status_pages` (monitor_ids/system_ids JSON columns — a join table for ≤ dozens of ids is ceremony); admin `/status`, public `/status/:slug` + `GET /api/status-pages/:slug/public`
- [x] Badges: `GET /api/badge/:kind/:id.svg` — stateless SVG from live data (monitor status+uptime, domain days-left, system status)
- [x] `GET /api/metrics` Prometheus exposition
- [x] Bulk: `POST /api/import/csv?kind=monitors|domains` header-driven CSV; monitors/domains/systems rows folded into `/api/export` (v2, additive)
- [x] ⌘K palette + header nav wired for new pages

## Phase 4 — Open-source launch → v1.0.0

- [ ] Docs: install, configuration, widget development guide (`docs/`)
- [ ] Importers: Homepage `services.yml`, Homarr JSON, Dashy `conf.yml` → sections/items/urls
- [ ] `README.md` hero: screenshots (4 themes), demo GIF, `docker run` one-liner
- [ ] goreleaser → binaries; GHCR image on tag; semver + `CHANGELOG.md`
- [ ] `CONTRIBUTING.md`, issue/PR templates, code of conduct, security policy
- [ ] i18n scaffolding (en default)
- [ ] Public repo flip + announcement assets

**Exit:** a stranger can `docker run`, import their Homepage config, and have a working board in under two minutes.

## Phase 5 — Post-1.0 (demand-driven only)

Build only what users actually ask for. Candidates, unordered:

- Optional auth (local password + OIDC) — most-requested, likely first
- Health-check history + uptime graphs
- Custom CSS / accent color / wallpaper
- Iframe/embed widgets, API-status widgets (generic JSON path)
- PWA install, mobile layout polish
- Multiple boards

## Non-goals (v1)

Auth, Postgres/external DB, multi-user, mobile app, container lifecycle
management, YAML-as-source-of-truth. Revisit only with evidence of demand.
