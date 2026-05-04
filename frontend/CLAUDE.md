# Dash Frontend – Claude Context

## Quick Reference
- Framework: Next.js 15 App Router (standalone output)
- Styling: Tailwind v4 + shadcn/ui + CSS custom properties for theming
- State: @tanstack/react-query (staleTime 30s)
- DnD: @dnd-kit/core + @dnd-kit/sortable
- API: openapi-fetch client generated from ../openapi/openapi.yaml
- Fonts: Geist Sans + Geist Mono (next/font/google)

## Theme System
3 themes via `data-theme` attribute on `<html>`:
- `light` — Vercel-inspired white
- `dark` — OLED black (default)
- `casaos` — Glassmorphism with backdrop-blur

## Key Paths
- `app/layout.tsx` — root layout with Providers
- `app/page.tsx` — renders DashboardPage
- `components/dashboard/dashboard-page.tsx` — main composition
- `lib/api/client.ts` — fetch wrapper for all API calls
- `lib/api/hooks.ts` — React Query hooks
- `lib/api/schema.ts` — TypeScript types (hand-written, matches OpenAPI)
