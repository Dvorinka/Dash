# Configuration

Dash is configured from the UI. The only environment the server understands:

| Variable | Default | Purpose |
|---|---|---|
| `DASH_PORT` | `8080` (`3000` in Docker) | listen port |
| `DASH_DATA_DIR` | `./data` (`/data` in Docker) | SQLite + uploaded icons |
| `DASH_DEV` | unset | `1` = dev mode: API only, no embedded UI |

## Data directory

```
data/
  dash.db        # SQLite — boards, monitors, domains, systems, ops
  icons/         # uploaded service icons
```

Back up the directory to back up everything. `/api/export` produces a JSON
snapshot; `/api/import/csv` bulk-loads monitors and domains from CSV.

## Notifications

**Settings → Notifications** holds one generic webhook URL
(`notify_webhook`). On monitor up/down transitions and domain events, Dash
POSTs a JSON payload with `event`, `name`, `text`, `content`, `message`, and
`at` fields — shaped so Slack, Discord, and ntfy all render it sensibly.
**Test** sends a `test` event. Delivery is async and non-fatal; failures are
logged.

## Monitors

HTTP(s), TCP, ping (ICMP), DNS, keyword, JSON-path, and push types. Per-monitor
interval, timeout, retries, headers/method/body, expected values, tags, and
notes. Push monitors expose `POST /api/push/:token` for cron-style jobs — the
monitor goes down if no push arrives within the expected window.

## Domains

WHOIS (RDAP first, TCP WHOIS fallback), DNS records, TLS certificate chain,
and provider detection run daily and on demand. Expiry and certificate
deadlines surface in the UI and through notifications.

## Systems

Bearer-token push ingest at `POST /api/systems/ingest`. A system is marked
offline when no sample arrives within a few intervals. Stats are pruned during
the hourly sweep.

## Operations

- **Incidents** — open/ack/resolve/close, with per-incident update timeline.
  A monitor going down opens an incident automatically; recovery resolves it.
- **Maintenance windows** — suppress monitor alerts for the window's scope.
- **Status pages** — public, unauthenticated views at `/status/:slug` listing
  selected monitors and systems with live state.
- **Badges** — `GET /api/badge/monitor/:id.svg` and
  `GET /api/badge/domain/:id.svg` for embedding status shields.
- **Metrics** — `GET /api/metrics` in Prometheus exposition format.

## Importers

Settings → Import accepts:

- Dash JSON export (full board restore)
- Homepage `services.yml`
- Homarr board JSON
- Dashy `conf.yml`
- CSV of monitors/domains (`/api/import/csv`)

Foreign formats map onto sections, items, and URLs; unknown fields are dropped.
