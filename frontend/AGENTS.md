# Dash Frontend – Agent Rules

## Scope
- This agent may edit: `/frontend`
- This agent must not edit `/backend`, `/db`, or `/openapi`

## Tech Stack
- Next.js 15 App Router + React 19 + TypeScript strict
- Tailwind CSS v4 + shadcn/ui (new-york style)
- `@tanstack/react-query` for server state
- `@dnd-kit` for drag-and-drop
- `openapi-typescript` + `openapi-fetch` for API client (generated from `../openapi/openapi.yaml`)

## Commands
- `npm run dev` — start dev server (Turbopack)
- `npm run build` — production build
- `npm run typecheck` — `tsc --noEmit`
- `npm run lint` — Next.js lint
- `npm run api:generate` — regenerate API types from OpenAPI spec

## Design
- Dark-first, Vercel-inspired aesthetic
- 3 themes: light, dark, casaos (glassmorphism)
- Geist Sans + Geist Mono fonts
- Shadow-as-border technique (no visible borders, use box-shadow)
- See `../Design.md` for full design system

## API Contract
- All types come from `../openapi/openapi.yaml`
- Do not invent contract fields outside OpenAPI
- API base URL: `NEXT_PUBLIC_API_BASE_URL` (default `http://localhost:8080`)

## Component Rules
- Use shadcn/ui primitives, do not rebuild from scratch
- All interactive elements must have focus rings
- Prefer `font-mono uppercase tracking-wide` for labels/badges
- Service cards are square aspect-ratio, icon + name + URL badges
- Groups are collapsible sections with chevron toggle
