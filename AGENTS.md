# Dash

Self-hosted homelab dashboard. Single Go binary serving an embedded React UI.
No auth, no YAML. See `ROADMAP.md` (canonical) and `docs/specs/` for design.

## Layout

- `apps/frontend` — React 18 + Vite + TS strict + Tailwind v4 + shadcn/ui
- `apps/backend` — Go + Gin + zap, CGO-free, SQLite via `modernc.org/sqlite`
- `packages/api-client` — generated TS client; `openapi.yaml` at repo root is the contract
- `infra/` — Dockerfile (multi-stage, one image)
- `tools/oxlint/anti-slop` — vendored lint plugin (do not extend casually)
- `data/` — runtime SQLite + uploaded icons (gitignored)

## Commands

```sh
npm install
npm run generate          # regenerate TS client from openapi.yaml
npm run dev               # vite dev server (proxies /api -> :8080)
npm run typecheck         # tsc --noEmit
npm run lint              # oxlint + anti-slop rules
npm run build             # api-client types + frontend bundle

cd apps/backend
go build ./...            # build backend
go test ./...             # tests
DASH_DEV=1 go run ./cmd/dash   # dev server on :8080, no embedded UI needed
```

## Rules

- OpenAPI is the API contract. Never hand-write API types in the frontend.
- All schema changes go through goose migrations in `apps/backend/internal/db/migrations`.
- Board renderers share state/dnd/popover primitives; never reimplement them per renderer.
- No emojis in code or UI.
- Before pushing: `npm run typecheck`, `npm run lint`, `npm run build`, `go build/vet/test ./...` all green.
