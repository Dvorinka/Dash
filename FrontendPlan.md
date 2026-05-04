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




# Dashboard Refactor & UX Plan

## Core Product Direction

The dashboard should feel empty, intentional, and flexible on first launch.

Current issue:

* The app starts with too much structure and too many assumptions.
* Users feel boxed into layouts before they build their own workspace.

New direction:

* Start with a clean canvas.
* Let users create widgets and apps only when needed.
* Prioritize drag-and-drop, layout freedom, responsiveness, and visual clarity.
* Make the dashboard feel closer to CasaOS in usability and visual hierarchy.

---

# 1. First Launch Experience

## Current Problem

Dashboard launches with widgets/groups already visible.

## New Behavior

On first launch:

* No widgets
* No pre-created services
* No placeholder cards
* No fake demo groups

Only show:

### Section 1 — Widgets

Top-right:

* Small `+ Add Widget` button

### Section 2 — Apps / Services

Top-right:

* Small `+ Add App` button

Layout example:

```text
Widgets ------------------------------------- [+]
(empty state)

Apps ---------------------------------------- [+]
(empty state)
```

## Empty State Design

Empty states should feel premium.

Example:

```text
No widgets yet
Create your first widget to customize your dashboard.
```

and:

```text
No apps added
Start by adding your first app or service.
```

Avoid giant centered buttons.

---

# 2. Layout Architecture

## Dashboard Sections

The dashboard should always contain:

1. Widgets Section
2. Apps Section

These are not groups.

These are permanent layout containers.

Groups belong inside Apps.

---

# 3. Widget System Improvements

## Current Problems

* Hard to resize
* Limited placement
* Dragging feels disconnected
* Drag icon placement is awkward
* Widgets feel static

## Required Improvements

### Fully Resizable Widgets

Users should be able to:

* Resize width
* Resize height
* Stretch across columns
* Fill entire section width
* Create masonry/grid layouts

Examples:

* Clock widget = small
* Pi-hole widget = large
* Analytics widget = full width

Recommended implementation:

### Use Grid-Based Resizing

Strong recommendation:

```text
react-grid-layout
```

Benefits:

* Resize handles
* Dragging support
* Collision detection
* Snap-to-grid
* Persistent positions
* Responsive layouts

---

## Widget Drag Handle

Current problem:

* Drag handle outside widget feels detached.

Fix:

* Drag handle should exist inside widget card.
* Top-right or top-left.

Example:

```text
[ Widget Title        ⋮⋮ ]
```

Users should instantly understand:

* drag
* settings
* resize

---

## Widget Responsiveness

Widgets must:

* Reflow on smaller screens
* Collapse naturally on mobile
* Maintain resize ratios
* Support multiple breakpoints

Recommended breakpoints:

```text
Desktop: 12-column grid
Tablet: 6-column grid
Mobile: 1-column stack
```

---

# 4. Clock Widget Improvements

## Current Problem

Timezone entry requires manual input.

## Better UX

Replace manual timezone input with:

### Searchable Dropdown

Recommended:

```text
Europe/Prague
Europe/London
America/New_York
Asia/Tokyo
```

### Better Option

Checkbox multi-select dropdown:

User can:

* Add multiple clocks
* Select timezone quickly
* Remove timezone instantly

Recommended libraries:

```text
react-select
shadcn Command + Popover
```

---

# 5. Widget Reliability

## Pi-hole Widget

### Must Validate:

* API reachable
* Token valid
* IP correct
* Live refresh updates
* Error states visible

Show:

```text
Cannot reach Pi-hole instance
Check URL or API key
```

---

## Memos Widget

### Must Validate:

Correct fields:

* API endpoint
* token/auth
* user scope
* response parsing

Must not silently fail.

---

## Refresh Button Issue

### Problem

Refresh button exists but cannot be clicked.

Likely causes:

* z-index overlap
* pointer-events disabled
* absolute layer blocking
* drag overlay intercepting clicks

Fix:

```css
pointer-events: auto;
z-index: 10;
```

Drag handles should not block interaction.

---

# 6. Drag & Drop — Highest Priority

## Core Principle

Drag-and-drop is the main feature.

It must feel effortless.

---

## Current Problems

* Dragging unreliable
* No placement preview
* Group movement broken
* Cross-group movement inconsistent

---

## Required Behavior

### Apps Should Be:

* Fully draggable
* Reorderable
* Group movable
* Cross-group movable
* Smooth animations

---

## Placement Preview

When dragging:

Show:

* Highlight insertion slot
* Ghost preview
* Position indicator

Users must know exactly where the app will land.

---

## Recommended Library

### Strong Recommendation

```text
@dnd-kit
```

Why:

* Best React drag library currently
* Excellent collision detection
* Smooth performance
* Group nesting support
* Sortable containers
* Keyboard accessible

---

## App Dragging Behavior

Allow:

```text
Ungrouped → Group
Group → Group
Group → Ungrouped
Reorder inside same group
Move group itself
```

Must be instant.

No modal.

No confirmation.

---

# 7. Apps Section Improvements

## Card View Problems

Current card view:

* Too rectangular
* Icon too small
* Doesn't feel visual enough

---

## New Card View

Make app cards square.

Inspired by CasaOS.

### New Card Structure

```text
┌───────────────┐
│               │
│     ICON      │
│               │
│  App Name     │
└───────────────┘
```

### Improvements

* Larger icons
* Centered content
* Better spacing
* More visual identity
* Hover interaction
* Rounded corners

Recommended:

```css
aspect-ratio: 1 / 1;
```

---

## List View

Keep mostly unchanged.

List view already works.

---

# 8. Groups System

## Problems

* Poor naming
* Not visually distinct
* Dragging unreliable
* Cannot collapse

---

## New Group Requirements

### Groups Should Support

* Expand/collapse
* Rename
* Drag reorder
* Nested app sorting
* Instant moving between groups

---

## Group Header Layout

```text
Infrastructure ▼           [⋮]
```

Avoid:

```text
GRP2
```

Groups must feel human.

---

## Group Dragging

Group itself should be draggable.

Move entire group section vertically.

---

# 9. Modal Improvements

## Problem

Modals have transparent backgrounds.

This reduces readability.

---

## Fix

Use proper modal surface.

Recommended:

```css
background: var(--surface);
backdrop-filter: blur(16px);
border: 1px solid rgba(255,255,255,.08);
```

No transparent forms.

---

# 10. Add App Flow

## Current Problem

Feels generic.

---

## Rename

Replace:

```text
Add Service
```

with:

```text
Add App
```

---

## Better Flow

### Modal Structure

Step 1:

* Choose app type

Step 2:

* Configure details

Step 3:

* Add icon/logo

Step 4:

* Select group

---

## Add App Card

When adding in-grid:

Small add tile.

Not giant button.

Example:

```text
+ Add App
```

Should visually match app cards.

---

# 11. URL Improvements

## Current Problem

URLs are visually weak.

---

## Better URL Display

Show:

```text
https://app.domain.com
```

With:

* favicon
* hostname extraction
* quick open
* copy button

Example:

```text
🌐 jellyfin.local
```

---

# 12. CasaOS-Inspired Theme

## Goal

Add an optional theme.

Not replacing current dark/light.

Add third style:

```text
CasaOS Inspired
```

---

## CasaOS Characteristics

### Visual Style

* Large spacing
* Rounded containers
* Soft shadows
* Glassmorphism feel
* Bigger cards
* Centered icons
* Calm background
* Floating panels

---

## CasaOS Dashboard Characteristics

### Keep

* App grid focus
* Icon-first navigation
* Background image
* Floating sections
* Minimal chrome

---

## Remove from CasaOS Reference

Do NOT include:

* Search bar
* Storage sync banner
* Drive discovery cards

Only use:

* Layout feel
* Card structure
* App sizing
* Background styling

---

## CasaOS Theme Structure

### Background

Use:

* gradient
* blurred wallpaper
* ambient overlay

---

### Panels

```css
background: rgba(18, 24, 40, 0.65);
backdrop-filter: blur(18px);
border-radius: 24px;
```

---

### App Cards

```css
aspect-ratio: 1;
border-radius: 28px;
transition: transform .2s ease;
```

Hover:

```css
transform: translateY(-3px);
```

---

## Theme Switcher

Add:

```text
Light
Dark
CasaOS Inspired
```

Store in:

```text
localStorage
```

---

# 13. Recommended Tech Stack Improvements

## Layout

```text
react-grid-layout
```

---

## Drag & Drop

```text
@dnd-kit
```

---

## Animations

```text
framer-motion
```

---

## UI Components

```text
shadcn/ui
```

---

## State

```text
zustand
```

---

# 14. Priority Order

## Phase 1 — Critical UX

1. Empty dashboard state
2. Widgets section + apps section
3. Smaller add buttons
4. Drag-and-drop fixes
5. Placement preview
6. Group movement
7. App movement between groups

---

## Phase 2 — Widget System

1. Resizable widgets
2. Widget drag handles
3. Better timezone picker
4. Fix refresh buttons
5. Widget validation

---

## Phase 3 — Visual Improvements

1. Square app cards
2. Better icon sizing
3. Modal redesign
4. Better group styling
5. URL redesign

---

## Phase 4 — CasaOS Theme

1. Theme architecture
2. Background system
3. Glass panels
4. CasaOS grid cards
5. Theme switcher

---

# 15. Biggest Product Rule

The dashboard should feel like:

* A workspace
* A customizable OS
* A clean home-lab control center
* A visual launcher
* Not a traditional admin panel

Users should instantly understand:

* Add
* Move
* Resize
* Organize
* Customize

without reading instructions.
