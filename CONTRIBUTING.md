# Contributing

Thanks for considering a contribution. Dash is a small, deliberate codebase —
read [AGENTS.md](AGENTS.md) and [ROADMAP.md](ROADMAP.md) first; the roadmap is
canonical for what gets built and when.

## Setup

Requires Go 1.24+ and Node 20+.

```sh
npm install
npm run generate       # regenerate the TS client from openapi.yaml
npm run dev            # vite dev server on :5173, proxies /api to :8080
```

```sh
cd apps/backend
DASH_DEV=1 go run ./cmd/dash   # API on :8080
```

## Rules of the house

- `openapi.yaml` is the API contract. Change it first, then
  `npm run generate`. Never hand-write API types in the frontend.
- All schema changes go through goose migrations in
  `apps/backend/internal/db/migrations` — `NNNN_name.sql` with `-- +goose Up`
  and `-- +goose Down`.
- Board renderers share state, drag-drop, and popover primitives. Never
  reimplement them per renderer.
- Auth is opt-in and local-only: no OIDC/SSO, no multi-user RBAC. No
  YAML-as-source-of-truth, no external database. These are non-goals.
- Standard library before dependencies; existing helpers before new ones.
- User-visible strings go through `t("key")` from `src/i18n` — add the key to
  `en.ts` rather than inlining literals in new UI.

## Before pushing

```sh
npm run typecheck
npm run lint
npm run build

cd apps/backend
go build ./... && go vet ./... && go test ./...
```

Plus a real smoke test of whatever you changed. All gates green or no push.

## Commits and PRs

- Conventional-ish subjects: `feat:`, `fix:`, `docs:`, `refactor:` — keep them
  short and describe the why.
- Small, reviewable PRs. Fill in the template; a screenshot of UI changes is
  worth more than a paragraph.

## Reporting bugs / requesting features

Use the issue templates. For security issues, see
[SECURITY.md](SECURITY.md) — do not open a public issue.
