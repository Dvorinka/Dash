# Dash — Design Spec & Roadmap

Date: 2026-09-16
Status: proposed — pending UI direction pick (see `mockups/`)

## 1. Product

Self-hosted homelab dashboard. A visual launchpad for services (local + external) with live widgets. CasaOS-grade ease of use, Vercel-grade design. Single Docker container, no auth in v1.

Differentiator: UI-first management (no YAML) + modern design. Homepage requires YAML, Homarr forces auth and complexity, Dashy is overloaded, CasaOS is an OS not a dashboard.

## 2. Requirements (from prompt.md)

| Req | Notes |
|---|---|
| Service cards | name, icon (URL or uploaded file), status indicator |
| Multi-URL services | each URL tagged `local` / `external` / custom label; 1 URL = direct nav, 2+ = popover chooser |
| Sections | named groups; collapse/expand; reorder sections and items via drag-drop; items move between sections |
| Add service | `+` button, dialog: name, URLs, icon |
| Widgets | live on the grid; v1: clock/timezones, Pi-hole, AdGuard Home, Immich; extensible registry |
| Themes | dark + light |
| No auth | v1; optional auth is post-1.0 |
| Lightweight | single container, single binary, <50 MB image target |

## 3. Architecture

```
/apps/frontend   React 18 + Vite + TS strict + Tailwind + shadcn/ui + dnd-kit
/apps/backend    Go + Gin + zap, CGO-free
/packages/api-client   generated from OpenAPI (single source of truth)
/infra           Dockerfile multi-stage → ONE image; docker-compose.yml (1 service)
/data            SQLite file + uploaded icons (bind-mounted)
```

### Approved deviations from locked stack

- **SQLite** (`modernc.org/sqlite`, CGO-free) instead of Postgres. Approved 2026-09-16. Reason: ~100 rows of state, single-binary deployment is the product's core install feature. goose migrations still apply.
- **No DragonflyDB.** Widget responses cached in-process with a small TTL map. Add a real cache only if profiling demands it.

Everything else per `tdvorak-fullstack` locked stack.

### Key decisions

- **Go embeds the frontend** via `embed.FS` → `docker run -v ./data:/data -p 3000:3000 dash` is the entire install.
- **Widget proxy server-side.** Pi-hole/AdGuard/etc. APIs need secrets and are not CORS-safe. Backend exposes `GET /api/widgets/:id/data`; a registry maps widget type → fetcher. New integration = one Go file + one React component.
- **Ordering** via fractional position column (`position REAL`) — drag-drop writes one row, not a renumber of the set.
- **Icons** stored in `data/icons/`; dashboard-icons pack integration for auto-suggest in Phase 3.

## 4. Data model (SQLite)

```sql
sections(id TEXT PK, name TEXT, position REAL, collapsed INT, created_at)
items(id TEXT PK, section_id FK, kind TEXT,      -- 'service' | 'widget'
      name TEXT, icon TEXT,                      -- icon: path or URL
      position REAL, config JSON, created_at)    -- widget cfg or null
urls(id TEXT PK, item_id FK, url TEXT, label TEXT, position REAL)
settings(key TEXT PK, value JSON)                -- theme, wallpaper, widget secrets ref
```

Widget secrets (API keys) live in `config JSON` for v1; encrypted-at-rest only if auth lands later.

## 5. API (OpenAPI contract — sketch)

```
GET/POST/PATCH/DELETE  /api/sections
POST                   /api/sections/reorder
GET/POST/PATCH/DELETE  /api/items
POST                   /api/items/reorder
POST                   /api/items/:id/icon       (upload)
GET                    /api/widgets/types        (registry metadata for UI)
GET                    /api/widgets/:id/data     (live fetch, cached ~30s)
GET/PUT                /api/settings
GET                    /api/export               (full JSON backup)
POST                   /api/import
GET                    /api/healthz
```

Frontend never hand-writes API types — generated client from `openapi.yaml`.

## 6. Frontend structure

- `ServiceCard` — icon, name, host, status dot, multi-URL badge → `UrlPopover`
- `Section` — header (label, count, collapse chevron, drag handle) + `SortableContext` grid
- `WidgetHost` — renders widget component by `config.type`, polls `/api/widgets/:id/data`
- `AddServiceDialog`, `EditServiceDialog` (shadcn Dialog + Form)
- `CommandPalette` (⌘K) — Phase 3
- dnd-kit `DndContext` at board level; sections and items both sortable

## 7. UI direction — pending

Static mockups in `mockups/` — open each, compare. Round 1 shortlist: A + B. C rejected (glass/wallpaper is not the target look).

| File | Direction | Character |
|---|---|---|
| `a-vercel-mono.html` | Flat monochrome | Geist, hairlines, no shadows, quietest |
| `b-command-bento.html` | Bento grid | mixed tile sizes, widgets inline, densest; bg glow disliked |
| ~~`c-casa-glass.html`~~ | ~~Glass + wallpaper~~ | rejected |
| `d-bento-mono.html` | Bento + mono palette | B's grid on A's zinc palette, no glow — direct merge of the two favorites |
| `e-editorial.html` | List paradigm | serif index headers, full-width rows, stat strip — no cards |
| `f-console.html` | Terminal | JetBrains Mono, bordered panels, `[ up ]` tags, prompt glyphs |

Decision recorded here once picked.

## 8. Roadmap

### Phase 0 — Foundation
- [ ] Monorepo scaffold, LICENSE (MIT), README, .gitignore, AGENTS.md
- [ ] `openapi.yaml` stub → generated TS client pipeline
- [ ] CI: `go build`, `go vet`, `tsc --noEmit`, oxlint + vendored anti-slop rules
- [ ] `docker compose up` runs the app

### Phase 1 — MVP
- [ ] Schema + goose migrations; section/item/URL CRUD
- [ ] Board UI: sections, cards, collapse, drag-drop ordering (persist position)
- [ ] Multi-URL popover; icon URL + upload
- [ ] Add/Edit dialogs; delete with confirm
- [ ] Dark/light; responsive to phone width
- [ ] Single-image Dockerfile; JSON export/import

### Phase 2 — Widgets
- [ ] Widget registry (Go) + `WidgetHost` (React)
- [ ] Clock/timezone widget (frontend-only)
- [ ] Pi-hole, AdGuard Home, Immich fetchers + per-widget config UI
- [ ] Service up/down ping (server-side HEAD, cached)

### Phase 3 — Open-source readiness
- [ ] Docs (install, config, widget dev guide), screenshots, demo GIF
- [ ] dashboard-icons auto-suggest in icon picker
- [ ] Importers: Homepage `services.yml`, Homarr JSON, Dashy `conf.yml`
- [ ] ⌘K search palette; i18n scaffolding
- [ ] goreleaser (binaries) + GHCR image + semver tags + CHANGELOG
- [ ] Issue/PR templates, CONTRIBUTING.md, code of conduct

### Phase 4 — Post-1.0 (demand-driven only)
- Optional auth (local + OIDC), health-history graphs, custom CSS/themes, iframe widgets, PWA, wallpaper support for direction A

### Explicit non-goals for v1
Auth, Postgres, multi-user, mobile app, container management, YAML-as-source-of-truth.

## 9. Testing

- Go: table-driven tests for widget fetchers + reorder logic (`go test ./...`)
- TS: `tsc --noEmit` + oxlint gate; dnd reducer unit tests only if ordering logic grows
- Smoke: `docker compose up` → create section → add service → drag → export JSON
