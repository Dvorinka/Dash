# Homelab Dashboard Project Plan

## Summary
Create `project.md` as full build spec for self-hosted homelab dashboard: Next.js React frontend, shadcn/ui, Go Gin backend, PostgreSQL via Docker Compose. Core v1: service cards, groups, drag/drop ordering, multi-URL launch picker, no auth, dark-first Vercel/Nothing-inspired UI, extensible widget framework with one real Pi-hole widget.

## Key Changes
- Scaffold monorepo:
  - `/frontend`: Next.js App Router, React, TypeScript strict, Tailwind, shadcn/ui.
  - `/backend`: Go + Gin API, zap logging, OpenAPI contract.
  - `/db`: PostgreSQL migrations with goose, typed queries with sqlc.
  - `/deploy`: Docker Compose for frontend, backend, Postgres.
- API source of truth: OpenAPI spec generates frontend TypeScript client/types.
- No authentication in v1; assume trusted LAN/self-hosted deployment.
- Persist:
  - services
  - service URLs marked `local` / `external` / custom label
  - icon URL or uploaded icon file reference
  - groups
  - group collapsed state
  - drag/drop ordering
  - dashboard widget instances/settings
- UI:
  - Dark-first, light mode also supported.
  - Vercel base: Geist Sans, Geist Mono, white/black neutrals, shadow-as-border, 6-8px radius.
  - Nothing accent: OLED dark mode, mono uppercase labels, red signal accent, sparse dot-grid detail.
  - Fonts loaded through `next/font/google`: `Geist`, `Geist_Mono`; optional `Doto` only for clock/hero widget if used.
- shadcn components:
  - Button, Card, Dialog, Sheet, Input, Select, Switch, Tabs, Badge, DropdownMenu, Tooltip, Collapsible, Alert, Empty, Separator, ScrollArea.
  - Drag/drop via `@dnd-kit`.
  - Forms use shadcn `FieldGroup` / `Field` patterns.

## Product Behavior
- Dashboard opens to time/date/timezone strip, widget row, grouped service grid.
- `+` action opens add-service dialog:
  - name
  - icon URL or uploaded icon
  - one or more URLs
  - URL label/type: local, external, custom
  - optional group
- Clicking service:
  - one URL: open directly
  - multiple URLs: show picker dialog, then open chosen URL
- Groups:
  - create, rename, delete empty group
  - collapse/expand
  - drag groups
  - drag services within and across groups
- Widgets:
  - v1 includes widget framework plus Pi-hole adapter.
  - Pi-hole widget stores base URL/API token locally in backend DB.
  - Widget failures render inline `[ERROR: ...]`, no toast spam.
  - Future adapters: AdGuard, Immich, custom image/status widgets.

## Test Plan
- Backend:
  - Go unit tests for service/group ordering, URL validation, widget config validation.
  - API integration tests against test Postgres.
  - Migration up/down check with goose.
- Frontend:
  - Component tests for service form, URL picker, group collapse.
  - Playwright flow: add service, add multiple URLs, choose URL, reorder cards, collapse group, toggle theme.
  - Accessibility check: keyboard focus, dialog titles, labels, contrast.
- Build:
  - `docker compose up` starts full stack.
  - OpenAPI generation succeeds.
  - Frontend typecheck and backend tests pass.

## Assumptions
- Use Next.js React, not SolidJS, because user wants to try Next.js and remain React-focused.
- Use PostgreSQL Docker, not SQLite, per chosen TDvorak default.
- Start visual design dark-first; light mode remains first-class.
- v1 ships no auth and is intended for trusted self-hosted network.
- `project.md` will be written with this plan when execution/mutation is allowed.
