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

## 7. UI system — DECIDED 2026-09-16

**The app ships multiple layouts, not one.** Same board state, same interactions (drag-drop, collapse, URL popover, add/edit), different structure per style. This is a core feature, not a theme afterthought.

**Primary renderer: `d-bento-mono.html`** — bento grid on the monochrome zinc palette. Geist + Geist Mono, hairline borders, no glow, status chips carry the only color (green/amber semantic).

**Full renderer set:**

| Renderer | Source mockup | Structure |
|---|---|---|
| Bento (default) | `d-bento-mono.html` | mixed-size grid, widgets inline as large tiles |
| Cards | `a-vercel-mono.html` | uniform compact card grid, widgets as cards |
| Index | `e-editorial.html` | serif section headers, full-width rows, masthead stat strip |
| Console | `f-console.html` | bordered panels/tables, JetBrains Mono, `[ up ]` tags |

Rejected and deleted: bento-with-glow (B), casa glass (C).

### Renderer contract (architectural requirement)

```
renderers/
  <name>/
    BoardView      (grid/list arrangement, section placement)
    SectionView    (header chrome: label, count, collapse, drag handle)
    ItemView       (tile/card/row anatomy: icon, name, host, status, popover anchor)
    WidgetView     (widget tile/panel/strip-cell anatomy)
```

- Board state (sections, items, order, collapse) lives in one store, renderer-agnostic.
- Interactions (dnd-kit, popover, dialogs) are provided by shared primitives the renderer composes — renderers never reimplement dnd or popover logic.
- Design tokens (`--bg`, `--surface`, `--border`, `--text`, …) are shared CSS vars; each renderer maps them to its own chrome. Dark/light stays orthogonal to renderer choice.
- Setting: `settings.renderer` — user picks layout in settings; persisted like theme.

Building the second renderer (Cards) inside Phase 1 is deliberate: it validates the seam while the code is still small.

## 8. Roadmap

Canonical roadmap lives in [`ROADMAP.md`](../../ROADMAP.md) at repo root — single source of truth, kept versioned with the code.

### Explicit non-goals for v1
Auth, Postgres, multi-user, mobile app, container management, YAML-as-source-of-truth.

## 9. Testing

- Go: table-driven tests for widget fetchers + reorder logic (`go test ./...`)
- TS: `tsc --noEmit` + oxlint gate; dnd reducer unit tests only if ordering logic grows
- Smoke: `docker compose up` → create section → add service → drag → export JSON
