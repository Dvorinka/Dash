# FrontendPlan.md

## Mission

Build the complete frontend for the self-hosted homelab dashboard. The frontend owns the Next.js app, shadcn UI composition, dashboard interactions, drag/drop UX, theme system, generated API client usage, frontend tests, and production-grade responsive design.

This agent may edit:

- `/frontend`

This agent must not edit `/backend`, `/db`, or `/openapi`. Read `openapi/openapi.yaml` only to generate client/types. If the API contract is missing or wrong, document the needed backend change instead of creating duplicate schemas.

## Product Context

The app is a fast, clean dashboard for home lab services. It should feel closer to Vercel, Linear, and a Nothing-style instrument panel than older dashboard tools. The first screen is the actual dashboard, not a landing page.

Core frontend responsibilities:

- Show date/time/timezones.
- Show dashboard widgets.
- Show grouped and ungrouped services.
- Add/edit/delete services with multiple URLs.
- Let user choose local/external/custom URL before opening when service has multiple URLs.
- Create/edit/collapse/delete groups.
- Drag/drop reorder groups, services, and widgets.
- Support dark and light mode.
- Use generated API types/client only.

## Stack

- Next.js App Router.
- React with TypeScript strict mode.
- Tailwind CSS.
- shadcn/ui.
- `@dnd-kit` for drag/drop.
- `@tanstack/react-query` for server state.
- `openapi-typescript` for generated types.
- `openapi-fetch` or a thin generated client wrapper for requests.
- Playwright for end-to-end tests.
- MSW or local fixtures while backend is incomplete.

## Repository Layout

Use this frontend layout:

```text
frontend/
  app/
    layout.tsx
    page.tsx
    globals.css
  components/
    dashboard/
    groups/
    services/
    widgets/
    shell/
    ui/
  lib/
    api/
    mocks/
    theme/
    utils/
  hooks/
  tests/
    e2e/
```

Intent:

- `app`: Next.js routes and global shell.
- `components/ui`: shadcn components only.
- `components/dashboard`: dashboard composition and layout.
- `components/groups`: group panels, group controls, group reorder.
- `components/services`: service cards, service form, URL picker.
- `components/widgets`: clock, image, Pi-hole widget cards, widget form.
- `lib/api`: generated client and React Query hooks.
- `lib/mocks`: fixtures/MSW handlers matching OpenAPI.
- `lib/theme`: theme constants and helpers.

## API Contract

Frontend consumes `../openapi/openapi.yaml`.

Generate types/client through scripts:

```json
{
  "scripts": {
    "api:generate": "openapi-typescript ../openapi/openapi.yaml -o lib/api/schema.ts",
    "typecheck": "tsc --noEmit",
    "test:e2e": "playwright test"
  }
}
```

Rules:

- Do not hand-write API response types.
- All API DTOs come from generated OpenAPI types.
- UI view models may exist, but must be derived from generated types.
- If backend is unavailable, use MSW fixtures shaped exactly like generated types.
- API base URL comes from `NEXT_PUBLIC_API_BASE_URL`, default `http://localhost:8080`.

Expected resources:

```text
Dashboard
Group
Service
ServiceUrl
WidgetInstance
WidgetData
AssetFile
ErrorResponse
```

Required frontend API hooks:

```text
useDashboard()
useCreateGroup()
useUpdateGroup()
useDeleteGroup()
useCreateService()
useUpdateService()
useDeleteService()
useUpdateLayout()
useUploadIcon()
useCreateWidget()
useUpdateWidget()
useDeleteWidget()
useWidgetData(widgetId)
useRefreshWidget(widgetId)
```

Mutation behavior:

- Optimistically update layout reorder where safe.
- Roll back reorder on API failure.
- For CRUD forms, close dialog only after success.
- Render inline errors as `[ERROR: message]`.
- Avoid toast dependency for core flows; inline status preferred.

## Design Direction

Primary design direction:

- Dark-first dashboard.
- Light mode complete and polished.
- Vercel-inspired structure: restrained cards, shadow-as-border, tight typography.
- Nothing-inspired accents: OLED black, mono uppercase labels, red signal accent only when meaningful.

Fonts:

- Load `Geist` and `Geist_Mono` with `next/font/google`.
- Optional `Doto` only for large clock/hero metric. Do not use Doto for body text.

Visual tokens:

- Background dark: near/OLED black.
- Background light: off-white/white.
- Text dark mode: white/soft gray hierarchy.
- Text light mode: `#171717` and neutral grays.
- Radius: 6px for controls, 8px for cards, no large rounded card stacks.
- Borders: use shadow-as-border or tokenized border, not heavy outlines.
- Accent: red only for alert/active signal/destructive state.
- Icons: monoline Lucide via shadcn project icon library.

Nothing/Vercel constraints:

- No gradients in app chrome.
- No decorative blobs/orbs.
- No emojis.
- No nested cards.
- No large marketing hero.
- No skeleton-only loading screens; prefer `[LOADING...]` or compact shadcn loading state.
- No visible instructional paragraphs explaining obvious UI.
- Use strong spacing and hierarchy instead of decorative containers.

## shadcn Rules

Use shadcn components before custom markup:

- Button
- Card
- Dialog
- Sheet
- Input
- Select
- Switch
- Tabs
- Badge
- DropdownMenu
- Tooltip
- Collapsible
- Alert
- Empty
- Separator
- ScrollArea
- Field / FieldGroup where available

Rules:

- Forms use `FieldGroup` and `Field` patterns.
- Dialogs, Sheets, and Drawers always have titles.
- Buttons with icons use `data-icon`.
- Use `gap-*`, never `space-x-*` or `space-y-*`.
- Use semantic Tailwind tokens, not random raw colors in component classes.
- Use `cn()` for conditional class names.
- Use shadcn `Empty` for no services/no groups.
- Use shadcn `Alert` for blocking errors.

## Screens and Components

### Dashboard Page

The first viewport is the dashboard.

Layout:

- Top shell with app name, current date/time, theme toggle, add button.
- Widget strip below header.
- Service area below widgets.
- Groups render as collapsible sections.
- Ungrouped services render in their own section when present.
- Empty dashboard renders a minimal empty state with primary add action.

Desktop:

- Max width around 1200-1400px.
- Dense but calm grid.
- Service cards use responsive columns.

Mobile:

- Single-column service grid.
- Add/edit forms use full-screen or near-full-height Sheet/Dialog.
- Drag handles must remain tappable.

### Service Cards

Show:

- Icon or generated initials fallback.
- Service name.
- URL kind badges or compact count.
- Optional primary URL label.
- Menu for edit/delete/move.

Click behavior:

- If service has one URL, open it directly in same tab or new tab based on app setting/default.
- If service has multiple URLs, open URL picker dialog.
- URL picker lists label, kind, and hostname.

Drag behavior:

- Drag service within group.
- Drag service across groups.
- Drag service to ungrouped.
- Show clear drop target state.
- Persist through `PUT /api/v1/layout`.

### Service Form

Fields:

- Name.
- Icon mode: icon URL or upload file.
- URLs dynamic list:
  - label
  - kind: local, external, custom
  - URL
  - primary toggle
- Group select.

Validation:

- Name required.
- At least one URL required.
- URL must be absolute `http` or `https`.
- Exactly one primary URL in UI; if user never chooses, first is primary.
- Show field-level errors, not generic banners only.

### Groups

Group section includes:

- Name.
- Service count.
- Collapse/expand control.
- Drag handle.
- Menu for rename/delete.

Behavior:

- Collapsed state persists via group PATCH.
- Empty group can be deleted.
- Non-empty group delete should ask user to move services to ungrouped, matching backend option.
- Group reorder persists through layout endpoint.

### Widgets

Widget strip supports:

- Clock widget.
- Image widget.
- Pi-hole widget.

Clock:

- Shows large time and date.
- Supports configured timezones when backend exposes config.
- Uses Geist Mono or optional Doto for display moment.

Image:

- Shows configured image in restrained card.
- Optional link click.
- No decorative cropping that hides important content.

Pi-hole:

- Shows status, blocked count, query count, percent blocked.
- Loading: `[LOADING...]`.
- Error: `[ERROR: ...]`.
- Stale data: show data with compact stale label.
- Manual refresh button.

Widget management:

- Add/edit widget Sheet.
- Widget type select.
- Type-specific config fields.
- Widget drag reorder persisted through layout endpoint.

## State Management

Use React Query for server state:

- Dashboard query is primary source for layout.
- Widget data can be separate query per widget.
- Invalidate dashboard after CRUD mutations.
- Use optimistic mutation for layout reorder.

Local UI state:

- Dialog open/closed.
- Drag active item.
- Theme.
- Form draft state.

Do not mirror entire dashboard into global client state unless needed for drag preview. If local reorder state is needed, keep it scoped to dashboard components and reconcile from query data.

## Accessibility

Required:

- Keyboard focus visible on all controls.
- Dialog title for every Dialog/Sheet.
- Service cards keyboard-accessible.
- Drag/drop has keyboard fallback or at least accessible controls for move up/down.
- Form fields have labels and error messages.
- Theme toggle has accessible label.
- Icon-only buttons have labels/tooltips.
- Color is not only signal for URL kind or errors.

## Testing Plan

Unit/component tests:

- Service form validates required name and URL.
- Dynamic URL rows add/remove correctly.
- URL picker opens for multi-URL service.
- Group collapse button updates state.
- Widget cards render loading/error/stale/data states.

Playwright tests:

- Dashboard loads fixture data.
- Add service with two URLs.
- Click multi-URL service and choose local/external URL.
- Create group and move service into group.
- Collapse group and confirm persisted state after reload.
- Drag reorder services and verify layout mutation.
- Toggle dark/light mode.
- Pi-hole widget shows data and refresh button.

Build checks:

```bash
npm run api:generate
npm run typecheck
npm run lint
npm run build
npm run test:e2e
```

Use the package manager chosen during app scaffold. If using pnpm, scripts remain same but commands run as `pnpm ...`.

## Implementation Steps

1. Scaffold Next.js App Router project in `/frontend` with TypeScript and Tailwind.
2. Initialize shadcn/ui and install required base components.
3. Add API generation script from `../openapi/openapi.yaml`.
4. Create generated client wrapper and React Query provider.
5. Build MSW/fixture data matching OpenAPI.
6. Implement theme tokens, fonts, dark/light modes, and shell.
7. Build dashboard page with fixture data.
8. Build service cards and URL picker.
9. Build add/edit service form with dynamic URLs and icon upload field.
10. Build group sections, collapse, menus, and group forms.
11. Add drag/drop for services, groups, and widgets.
12. Build widgets: clock, image, Pi-hole.
13. Wire all CRUD and layout mutations to API client.
14. Add tests and responsive polish.
15. Run build/typecheck/e2e.

## Acceptance Criteria

- `/frontend` builds successfully.
- Frontend generates types from `../openapi/openapi.yaml`.
- No hand-written API DTO duplication.
- Dashboard works with MSW fixtures before backend is ready.
- Dashboard works against backend API once available by changing `NEXT_PUBLIC_API_BASE_URL`.
- Add/edit/delete service flows work.
- Multiple URL picker works.
- Groups can be created, collapsed, reordered, and deleted safely.
- Drag/drop persists through layout API.
- Clock, image, and Pi-hole widgets render expected states.
- Dark and light modes are polished.
- Mobile and desktop layouts do not overlap or clip text.

## Parallel Coordination

- Frontend can start with fixtures while backend builds API.
- Do not invent contract fields outside OpenAPI.
- If UI needs a new field, request OpenAPI change from backend agent.
- Regenerate API client after any OpenAPI update.
- Keep all frontend work isolated inside `/frontend`.

