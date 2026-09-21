# Widget development

A widget is two pieces:

1. a Go **fetcher** in `apps/backend/internal/widget/` that declares config
   fields and pulls live data, and
2. a React **view** in `apps/frontend/src/widgets/` that renders the payload
   inside a tile.

Renderers own the tile chrome — widgets never touch layout, drag-drop, or
popovers.

## Backend fetcher

One file, registered in `init()`:

```go
package widget

type pihole struct{}

func init() { Register(pihole{}) }

func (pihole) Meta() Type {
	return Type{
		Type:        "pihole",
		Name:        "Pi-hole",
		Description: "Queries and block rate",
		Fields: []Field{
			{Key: "endpoint", Label: "Endpoint", Placeholder: "http://pi.hole", Required: true},
			{Key: "token", Label: "API token", Secret: true},
		},
	}
}

func (pihole) Fetch(ctx context.Context, cfg json.RawMessage) (any, error) {
	// cfg holds the item's config JSON. Never log it — it may hold keys.
}
```

- `Meta()` drives the add-widget dialog: `GET /api/widgets/types` returns
  every registered type with its fields.
- `Fetch` is called by `GET /api/widgets/:id/data`; responses are cached ~30s
  in-process. Return any JSON-marshalable value — it reaches the frontend as
  `data`.
- Read config with the `cfgStr`/`cfgNum` helpers in the package.
- Fetchers needing the database (monitor, domain, system) live in
  `internal/api` and register via `widget.Register` at router setup — the
  `widget` package stays a leaf with no DB dependency.

## Frontend view

One component in `apps/frontend/src/widgets/`, one line in the registry at
`src/widgets/index.tsx`:

```tsx
export function PiholeWidget({ data, error }: { item: Item; data?: WidgetData; error?: string }) {
	if (error) return <TileError msg={error} />;
	if (!data) return <TileLoading />;
	return <div>{data.blockedPercent}%</div>;
}
```

```ts
const registry: WidgetRegistry = {
	pihole: { view: PiholeWidget },
};
```

- `useWidgetData` polls `/api/widgets/:id/data` every 30s. `data` is the
  fetcher's payload, `error` is a human-readable upstream failure.
- Payloads are free-form; read fields defensively (values may be absent on
  first paint).
- `local: true` marks a widget as frontend-only (e.g. clock) — no fetcher, no
  polling.

## Config fields

`Fields` in `Meta()` render inputs in the add-widget dialog. `Secret: true`
masks the input. Config is stored in `items.config` JSON, sent back to the
fetcher on every poll, and never logged.

## Checklist

- [ ] `apps/backend/internal/widget/<name>.go` — fetcher + `init()` + `Meta()`
- [ ] `apps/frontend/src/widgets/<name>.tsx` — view component
- [ ] registry entry in `src/widgets/index.tsx`
- [ ] error and empty-data states render cleanly in the tile
- [ ] upstream timeouts bounded (respect `ctx`, keep HTTP client timeouts)
