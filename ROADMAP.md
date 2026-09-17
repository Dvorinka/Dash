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
| `v0.1.0` | Phase 0 + 1 — working MVP, single image | manual smoke test |
| `v0.2.0` | Phase 2 — widget layer + first integrations | Pi-hole widget live |
| `v0.3.0` | Phase 3 — alternate themes, search, icon packs | theme switcher works |
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
- [x] `docker-compose.yml` — one service, `./data:/data` volume

**Exit:** `docker compose up` serves a blank board — verified locally (33.6 MB image). CI green on `main` — pending first push.

## Phase 1 — MVP → v0.1.0

The product's spine: sections, services, drag-drop, multi-URL.

- [ ] Schema: `sections`, `items`, `urls`, `settings` (fractional `position REAL` ordering)
- [ ] REST API: CRUD for sections/items/urls, `reorder` endpoints, icon upload, `settings`, `export`/`import` JSON
- [ ] Renderer contract: shared board store + `BoardView`/`SectionView`/`ItemView`/`WidgetView` seam, dnd + popover as shared primitives
- [ ] **Bento** renderer (default): mixed-size grid, hairline section headers, tiles with status chips
- [ ] **Cards** renderer: uniform compact grid — second renderer proves the seam while code is small
- [ ] dnd-kit: reorder items in a section, move items between sections, reorder sections; persist on drop
- [ ] Collapse/expand sections (persisted)
- [ ] `UrlPopover`: 1 URL → direct open, 2+ → chooser (`local`/`external`/custom tags)
- [ ] `AddServiceDialog` / `EditServiceDialog` (name, URLs+labels, icon URL or file upload)
- [ ] Icon handling: upload to `data/icons/`, remote URL passthrough, letter-tile fallback
- [ ] Dark/light toggle (persisted), renderer picker in settings, responsive to ~360px
- [ ] Status ping: server-side HEAD request per service URL, cached ~60s → `up`/`down` chip

**Exit:** create sections, add services with icons and dual URLs, drag everything, reload — state persists. `docker run` single image works. Export/import round-trips.

## Phase 2 — Widgets → v0.2.0

- [ ] Go widget registry: `Widget` interface (`Type()`, `Fetch(cfg)`), `GET /api/widgets/:id/data` with ~30s in-process TTL cache
- [ ] `GET /api/widgets/types` → registry metadata drives the add-widget dialog
- [ ] `WidgetHost` component: maps `config.type` → React component, polls with backoff
- [ ] Widgets as grid items (same `items` table, `kind = 'widget'`, `config JSON`)
- [ ] Clock/timezone widget (frontend-only)
- [ ] Pi-hole fetcher (blocked %, queries today, per-day bar series)
- [ ] AdGuard Home fetcher (blocked %, clients)
- [ ] Immich fetcher (photo count, library size)
- [ ] Per-widget config UI (endpoint URL, API key — stored in `config`, never logged)

**Exit:** Pi-hole widget shows real data; adding a new integration = one Go file + one React component.

## Phase 3 — Renderers & ergonomics → v0.3.0

- [ ] **Index** renderer: serif section headers, full-width rows, masthead stat strip
- [ ] **Console** renderer: bordered panels/tables, JetBrains Mono, `[ up ]` tags
- [ ] Renderer switcher polished (instant swap, per-renderer preview in settings)
- [ ] ⌘K command palette: jump to service, add service, toggle theme, switch renderer
- [ ] dashboard-icons auto-suggest in icon picker (name → icon URL)
- [ ] User services list → prioritized widget backlog (owner-supplied list pending)

**Exit:** all four renderers work off the same board state; palette navigates everything.

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
