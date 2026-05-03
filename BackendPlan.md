# BackendPlan.md

## Mission

Build the complete backend for the self-hosted homelab dashboard. The backend owns persistence, OpenAPI, migrations, ordering rules, icon uploads, widget integrations, Docker runtime, and production-grade API behavior.

This agent may edit:

- `/backend`
- `/db`
- `/openapi`
- root infra/docs needed for backend integration, such as `docker-compose.yml`, `.env.example`, `Makefile`, and `README.md`

This agent must not edit `/frontend` except to document commands or contract expectations.

## Product Context

The app is a lightweight dashboard for managing self-hosted and external services. It replaces older dashboard-style tools with a modern Vercel/Nothing-inspired interface. v1 has no authentication and assumes trusted LAN/self-hosted use.

Core backend responsibilities:

- Store service cards and multiple launch URLs per service.
- Store groups, collapsed state, and drag/drop ordering.
- Store dashboard widgets and widget configuration.
- Provide one real third-party widget adapter: Pi-hole.
- Store uploaded service icons locally.
- Serve a stable OpenAPI contract so frontend can generate types/client.

## Stack

- Go 1.22 or newer.
- Gin for HTTP routing.
- zap for structured logging.
- pgx for PostgreSQL access.
- PostgreSQL 16.
- goose for migrations.
- sqlc for typed query generation.
- OpenAPI 3.1 in `openapi/openapi.yaml`.
- Docker Compose for local/self-hosted runtime.

## Repository Layout

Backend implementation should use this structure:

```text
backend/
  cmd/server/
    main.go
  internal/config/
  internal/http/
  internal/services/
  internal/store/
  internal/widgets/
  internal/assets/
  internal/validation/
  internal/testutil/
db/
  migrations/
  query/
openapi/
  openapi.yaml
```

Package intent:

- `cmd/server`: process entrypoint, config load, logger, DB pool, router start.
- `internal/config`: env parsing, defaults, validation.
- `internal/http`: routes, handlers, request/response mapping.
- `internal/services`: business rules and ordering transactions.
- `internal/store`: sqlc-generated store plus thin transaction helpers.
- `internal/widgets`: widget registry, Pi-hole adapter, cache refresh logic.
- `internal/assets`: icon upload validation and local file storage.
- `internal/validation`: shared URL/name/config validation.

## Runtime Configuration

Provide `.env.example` with:

```env
APP_ENV=development
HTTP_ADDR=:8080
DATABASE_URL=postgres://dash:dash@postgres:5432/dash?sslmode=disable
DATA_DIR=/data
PUBLIC_BASE_URL=http://localhost:8080
WIDGET_FETCH_TIMEOUT=5s
WIDGET_CACHE_TTL=60s
MAX_ICON_UPLOAD_BYTES=524288
ALLOWED_ORIGINS=http://localhost:3000
```

Rules:

- Fail fast on invalid required config.
- Create icon upload directory under `DATA_DIR/icons`.
- Use CORS only for configured origins in development/self-hosted split runtime.
- Keep all secrets in env or database, never hardcoded.

## Data Model

Use UUID primary keys, `created_at`, `updated_at`, and integer `sort_order` where order matters.

Tables:

```text
groups
  id uuid pk
  name text not null
  sort_order int not null
  collapsed boolean not null default false
  created_at timestamptz not null
  updated_at timestamptz not null

services
  id uuid pk
  group_id uuid null references groups(id) on delete set null
  name text not null
  icon_url text null
  icon_asset_id uuid null references asset_files(id) on delete set null
  sort_order int not null
  created_at timestamptz not null
  updated_at timestamptz not null

service_urls
  id uuid pk
  service_id uuid not null references services(id) on delete cascade
  label text not null
  kind text not null check (kind in ('local', 'external', 'custom'))
  url text not null
  sort_order int not null
  is_primary boolean not null default false
  created_at timestamptz not null
  updated_at timestamptz not null

widget_instances
  id uuid pk
  type text not null check (type in ('clock', 'image', 'pihole'))
  title text not null
  enabled boolean not null default true
  sort_order int not null
  config jsonb not null default '{}'
  created_at timestamptz not null
  updated_at timestamptz not null

widget_cache
  widget_id uuid pk references widget_instances(id) on delete cascade
  status text not null check (status in ('fresh', 'stale', 'error'))
  data jsonb null
  error text null
  fetched_at timestamptz null
  expires_at timestamptz null
  updated_at timestamptz not null

asset_files
  id uuid pk
  original_name text not null
  stored_name text not null
  mime_type text not null
  size_bytes int not null
  public_path text not null
  created_at timestamptz not null
```

Indexes:

- `groups(sort_order)`
- `services(group_id, sort_order)`
- `service_urls(service_id, sort_order)`
- `widget_instances(enabled, sort_order)`
- `asset_files(created_at)`

Ordering invariant:

- Groups order independently.
- Services order inside their current `group_id`; ungrouped services use `group_id = null`.
- Widgets order independently.
- Reorder endpoints accept full ordered lists and persist in one transaction.

## OpenAPI Contract

Backend owns `openapi/openapi.yaml`. Create/update it before handler implementation. Frontend must be able to generate TypeScript client/types from it.

Base path:

```text
/api/v1
```

Health:

```text
GET /health
```

Required endpoints:

```text
GET    /api/v1/dashboard

GET    /api/v1/groups
POST   /api/v1/groups
GET    /api/v1/groups/{groupId}
PATCH  /api/v1/groups/{groupId}
DELETE /api/v1/groups/{groupId}

GET    /api/v1/services
POST   /api/v1/services
GET    /api/v1/services/{serviceId}
PATCH  /api/v1/services/{serviceId}
DELETE /api/v1/services/{serviceId}

PUT    /api/v1/layout

POST   /api/v1/assets/icons

GET    /api/v1/widgets
POST   /api/v1/widgets
GET    /api/v1/widgets/{widgetId}
PATCH  /api/v1/widgets/{widgetId}
DELETE /api/v1/widgets/{widgetId}
GET    /api/v1/widgets/{widgetId}/data
POST   /api/v1/widgets/{widgetId}/refresh
```

Core response schemas:

```yaml
Dashboard:
  groups: Group[]
  ungroupedServices: Service[]
  widgets: WidgetInstance[]

Group:
  id: string
  name: string
  sortOrder: integer
  collapsed: boolean
  services: Service[]
  createdAt: string
  updatedAt: string

Service:
  id: string
  groupId: string | null
  name: string
  iconUrl: string | null
  iconAssetId: string | null
  sortOrder: integer
  urls: ServiceUrl[]
  createdAt: string
  updatedAt: string

ServiceUrl:
  id: string
  label: string
  kind: local | external | custom
  url: string
  sortOrder: integer
  isPrimary: boolean

WidgetInstance:
  id: string
  type: clock | image | pihole
  title: string
  enabled: boolean
  sortOrder: integer
  config: object
  createdAt: string
  updatedAt: string
```

Error schema:

```yaml
ErrorResponse:
  code: string
  message: string
  details: object | null
```

Use stable error codes:

- `validation_error`
- `not_found`
- `conflict`
- `upload_too_large`
- `unsupported_media_type`
- `widget_fetch_failed`
- `internal_error`

## API Behavior

Dashboard:

- `GET /dashboard` returns all enabled/visible dashboard data in render order.
- Include services nested under groups.
- Include ungrouped services separately.
- Include widgets regardless of data freshness; data comes from widget data endpoint.

Groups:

- Create group with next `sort_order`.
- Rename and collapsed state via PATCH.
- Delete empty group by default.
- If group contains services, return `409 conflict` unless request includes explicit handling:
  - `?moveServicesToUngrouped=true` moves services to ungrouped.
  - No hard delete of child services in v1.

Services:

- Create/update accepts service fields and full URL list.
- A service must have at least one URL.
- URL labels must be non-empty.
- Exactly one primary URL is allowed; if omitted, first URL becomes primary.
- Patch replaces URL list atomically for simplicity.
- Delete service cascades URLs.

Layout:

- `PUT /layout` accepts ordered groups, ordered widgets, and ordered services per group/ungrouped.
- Validate all referenced IDs exist.
- Validate every submitted service appears once.
- Persist all ordering in one DB transaction.
- Return updated dashboard.

Assets:

- `POST /assets/icons` accepts multipart field `file`.
- Allow PNG, JPEG, WebP, SVG only.
- Enforce `MAX_ICON_UPLOAD_BYTES`.
- Store file under `DATA_DIR/icons`.
- Return `AssetFile` with `publicPath`.
- Serve uploaded icons from `/uploads/icons/{storedName}`.

Widgets:

- Widget config is JSONB but validated by widget type.
- `clock` config: `{ "timezones": string[] }`.
- `image` config: `{ "imageUrl": string, "linkUrl": string | null }`.
- `pihole` config: `{ "baseUrl": string, "apiToken": string }`.
- Do not expose `apiToken` in normal widget responses. Return masked/omitted secret fields.
- `GET /widgets/{id}/data` returns cache if fresh; refreshes if expired where reasonable.
- `POST /widgets/{id}/refresh` forces fetch and updates cache.

Pi-hole adapter:

- Use HTTP client with timeout from config.
- Normalize base URL and call Pi-hole summary endpoint.
- Support modern Pi-hole API where possible, but isolate endpoint details behind adapter.
- Return compact data useful for cards: blocked count, query count, percent blocked, status, fetchedAt.
- On failure, keep stale data if available and return status `stale` with error.
- If no data exists, return status `error` with `[ERROR: ...]` compatible message.

## Validation Rules

- Names: trim whitespace, 1 to 80 chars.
- Labels: trim whitespace, 1 to 40 chars.
- URLs: absolute `http` or `https` only for service URLs and widget URLs.
- Icon URL: absolute `http`/`https` or uploaded asset reference.
- Reject duplicate service URL IDs in update payload.
- Reject unknown widget type.
- Reject unknown fields only if OpenAPI decoder supports this cleanly; otherwise ignore safely.

## Security Posture

v1 has no authentication. Still implement baseline hardening:

- No arbitrary file paths from user input.
- Uploaded file names must be generated server-side.
- SVG uploads must be served as files, not inlined.
- Do not log Pi-hole tokens.
- CORS restricted to configured origins.
- Use request body size limits.
- Return generic internal errors to clients.

## Docker and Local Run

Provide root `docker-compose.yml` with:

- `postgres`
- `backend`

Frontend service may be added by frontend agent later. Backend compose must still work alone.

Backend container:

- Multi-stage Dockerfile.
- Runs migrations before server start, or provide clear `make migrate-up` used by compose entrypoint.
- Exposes `8080`.
- Mounts persistent data volume for `/data`.

Useful commands:

```bash
make backend-dev
make backend-test
make db-migrate-up
make db-migrate-down
make openapi-validate
make sqlc
docker compose up
```

## Implementation Steps

1. Scaffold Go module and backend package layout.
2. Create Docker Compose, backend Dockerfile, `.env.example`, Makefile targets.
3. Write initial goose migrations for all tables/indexes.
4. Write sqlc config and queries for CRUD, dashboard loading, and ordering.
5. Write `openapi/openapi.yaml` with all schemas/endpoints/errors.
6. Implement config, logger, DB pool, health route.
7. Implement group/service/dashboard APIs.
8. Implement layout transaction.
9. Implement asset upload and static file serving.
10. Implement widget registry, config validation, cache table logic.
11. Implement Pi-hole adapter and refresh/data endpoints.
12. Add backend tests and integration tests.
13. Run full backend validation.

## Test Plan

Unit tests:

- URL validation accepts `http`/`https`, rejects relative and unsupported schemes.
- Service create/update requires at least one URL.
- Primary URL fallback selects first URL.
- Group delete rejects non-empty group without move flag.
- Widget config validation rejects missing Pi-hole token/base URL.
- Pi-hole token masking never returns raw token.

Integration tests:

- Migrations apply cleanly to empty Postgres.
- Create group, create service with two URLs, fetch dashboard.
- Move service between groups through layout endpoint.
- Reorder groups/services/widgets in one transaction.
- Delete service cascades URLs.
- Upload valid icon, reject oversized/unsupported file.
- Widget refresh caches success and preserves stale data on later failure.

Smoke tests:

```bash
go test ./...
go vet ./...
make openapi-validate
docker compose up --build
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/dashboard
```

## Acceptance Criteria

- `openapi/openapi.yaml` exists and is valid.
- Backend compiles with `go test ./...`.
- Migrations run against PostgreSQL 16.
- API returns deterministic JSON matching OpenAPI.
- Dashboard data persists after container restart.
- `docker compose up` boots backend and Postgres cleanly.
- Frontend agent can generate client from OpenAPI without manual type fixes.

## Parallel Coordination

- Backend can proceed before frontend exists.
- Keep breaking OpenAPI changes deliberate and documented.
- If contract changes, update `openapi/openapi.yaml` first, then backend handlers/tests.
- Do not create frontend types by hand.
- Do not move shared contract into frontend.

