# Dash — Roadmap

A self-hosted homelab dashboard. CasaOS-grade ease of use, Vercel-grade design.
Single container, UI-managed, no YAML, no auth required.

Stack: **Go + Gin + SQLite** (single static binary, embeds frontend) · **React 18 + Vite + TS strict + Tailwind + shadcn/ui + dnd-kit** · OpenAPI-generated client.

---

## Design system

Primary renderer: **Bento Mono** (`mockups/d-bento-mono.html`) — bento grid on a
monochrome zinc palette, Geist + Geist Mono, hairline borders, semantic status
color only.

Alternate view themes ship post-MVP on the same data model (Phase 3.5):

| Theme | Source mockup | Character |
|---|---|---|
| Bento Mono (default) | `d-bento-mono.html` | mixed-size grid, widgets inline |
| Mono Cards | `a-vercel-mono.html` | uniform card grid, quietest |
| Index | `e-editorial.html` | serif headers, list rows, stat strip |
| Console | `f-console.html` | terminal panels, `[ up ]` tags |

Rule: themes change the renderer, never the data or the interactions.

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

## Phase 0 — Foundation → part of v0.1.0

Repo hygiene and pipelines before features.

- [ ] Monorepo scaffold (`apps/frontend`, `apps/backend`, `packages/api-client`, `infra/`, `data/`)
- [ ] MIT `LICENSE`, `README.md`, `.gitignore`, `AGENTS.md`
- [ ] `openapi.yaml` stub → generated TS client (`packages/api-client`)
- [ ] Backend skeleton: Gin router, `zap` logging, `/api/healthz`, SQLite open + goose migrations runner
- [ ] Frontend skeleton: Vite + TS strict + Tailwind + shadcn/ui, theme tokens matching `d-bento-mono.html` CSS vars
- [ ] CI (GitHub Actions): `go build ./...`, `go vet ./...`, `go test ./...`, `tsc --noEmit`, `oxlint` + vendored anti-slop rules, `npm run build`
- [ ] Multi-stage `Dockerfile` (frontend build → Go build → `FROM scratch`/distroless, `embed.FS` serves UI)
- [ ] `docker-compose.yml` — one service, `./data:/data` volume

**Exit:** `docker compose up` serves a blank board. CI green on `main`.

## Phase 1 — MVP → v0.1.0

The product's spine: sections, services, drag-drop, multi-URL.

- [ ] Schema: `sections`, `items`, `urls`, `settings` (fractional `position REAL` ordering)
- [ ] REST API: CRUD for sections/items/urls, `reorder` endpoints, icon upload, `settings`, `export`/`import` JSON
- [ ] Board UI (Bento Mono): section headers (label, count, hairline, chevron), service tiles (icon, name, host, status chip)
- [ ] dnd-kit: reorder items in a section, move items between sections, reorder sections; persist on drop
- [ ] Collapse/expand sections (persisted)
- [ ] `UrlPopover`: 1 URL → direct open, 2+ → chooser (`local`/`external`/custom tags)
- [ ] `AddServiceDialog` / `EditServiceDialog` (name, URLs+labels, icon URL or file upload)
- [ ] Icon handling: upload to `data/icons/`, remote URL passthrough, letter-tile fallback
- [ ] Dark/light toggle (persisted), responsive to ~360px
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

## Phase 3 — Themes & ergonomics → v0.3.0

- [ ] Theme token layer finalized; view-theme switcher in settings
- [ ] Alternate renderers: Mono Cards, Index, Console
- [ ] ⌘K command palette: jump to service, add service, toggle theme
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
