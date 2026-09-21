# Changelog

All notable changes to Dash. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.7.0] — 2026-09-21

### Added

- Incidents with open/ack/resolve/close lifecycle and per-incident update
  timeline; monitors auto-open and auto-resolve incidents on transitions
- Maintenance windows that suppress monitor alerts
- Public status pages at `/status/:slug` (unauthenticated, per-page monitor
  and system selection)
- SVG badges: `/api/badge/monitor/:id.svg`, `/api/badge/domain/:id.svg`
- Prometheus metrics at `/api/metrics`
- CSV bulk import (`/api/import/csv`) and monitors/domains/systems in JSON
  export
- Generic webhook notifications (`notify_webhook` setting): monitor up/down
  and domain events, Slack/Discord/ntfy-compatible payload, `/api/notify/test`

## [0.6.0] — 2026-09-21

### Added

- System monitoring: `systems` + `system_stats`, bearer-token push ingest
  (`/api/systems/ingest`), offline detection
- `dash-agent` binary: Linux collector for CPU, memory, swap, disk, network,
  load, uptime, temperatures, and Docker container state
- Systems list and detail pages with Recharts graphs; system board widget

## [0.5.0] — 2026-09-21

### Added

- Domain watch: RDAP + TCP-WHOIS lookup, WHOIS parsing, DNS records
  (NS/MX/TXT/A/AAAA), TLS certificate details, host geo, DNS/email/hosting/
  certificate provider detection
- Daily refresh scheduler, manual refresh, check history
- Domain list and detail pages, expiry and SSL countdowns, domain board widget

## [0.4.0] — 2026-09-21

### Added

- Uptime monitors: HTTP, TCP, ping, DNS, keyword, JSON-path, and push types;
  interval/timeout/retries config, heartbeat history, 24h/30d uptime stats
- Monitor scheduler, check-now endpoint, push ingest (`/api/push/:token`)
- Monitor list and detail pages, monitor board widget, service-to-monitor
  creation checkbox
- Importers: Homepage `services.yml`, Homarr JSON, Dashy `conf.yml`

## [0.3.0] — 2026-09-17

### Added

- Index and Console board renderers (four total), command palette (⌘K),
  icon CDN suggestions

## [0.2.0] — 2026-09-17

### Added

- Widget layer: Go fetcher registry, config-field metadata, 30s TTL cache
- Pi-hole, AdGuard Home, Immich, and clock widgets; per-widget config UI

## [0.1.0] — 2026-09-17

### Added

- Board MVP: sections, items, multi-URL services, drag-drop, collapse state
- Bento and Cards renderers, dark/light themes, icon upload + letter fallback
- Server-side status pings with up/down chips
- Single Go binary serving the embedded React UI; SQLite via goose migrations;
  Docker image on distroless

[Unreleased]: https://github.com/Dvorinka/Dash/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/Dvorinka/Dash/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/Dvorinka/Dash/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/Dvorinka/Dash/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/Dvorinka/Dash/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/Dvorinka/Dash/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/Dvorinka/Dash/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Dvorinka/Dash/releases/tag/v0.1.0
