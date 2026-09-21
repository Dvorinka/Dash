# Dash

A self-hosted homelab dashboard. CasaOS-grade ease of use, Vercel-grade design.
Single container, UI-managed, no YAML, no auth.

![Dash board](docs/screenshots/board.png)

## Features

- **Service board** — sections, drag-and-drop, multi-URL services, icon upload,
  four board renderers (Bento, Cards, Index, Console), dark/light themes
- **Uptime monitors** — HTTP, TCP, ping, DNS, keyword, JSON-path, and push
  checks on a schedule, with heartbeat history and uptime stats
- **Domain watch** — WHOIS/RDAP, DNS records, TLS certificate expiry,
  registrar and provider detection
- **System monitoring** — lightweight `dash-agent` pushes CPU, memory, disk,
  network, load, temperatures, and Docker container state
- **Incidents & status pages** — manual and automatic incidents, maintenance
  windows, public `/status/:slug` pages, SVG badges
- **Alerts** — generic webhook notifications (Slack, Discord, ntfy compatible)
- **Observability** — Prometheus `/api/metrics`, JSON export, CSV bulk import,
  importers for Homepage, Homarr, and Dashy configs

## Run

```sh
docker run -d -v ./data:/data -p 3000:3000 ghcr.io/dvorinka/dash:latest
```

or with compose:

```yaml
services:
  dash:
    image: ghcr.io/dvorinka/dash:latest
    ports:
      - "3000:3000"
    volumes:
      - ./data:/data
    restart: unless-stopped
```

Then open http://localhost:3000. Everything is configured from the UI.

## System agent

Monitor a Linux host: create a system in the UI, copy the token, then:

```sh
dash-agent -url http://dash:3000 -token <token>
```

One static binary, no dependencies. See
[docs/installation.md](docs/installation.md) for systemd and Docker setups.

## Screenshots

| | |
|---|---|
| ![Monitors](docs/screenshots/monitors.png) | ![Domains](docs/screenshots/domains.png) |
| ![Systems](docs/screenshots/systems.png) | ![Status page](docs/screenshots/status-page.png) |

## Docs

- [Installation](docs/installation.md)
- [Configuration](docs/configuration.md)
- [Widget development](docs/widgets.md)
- [Roadmap](ROADMAP.md) · [Changelog](CHANGELOG.md) · [Contributing](CONTRIBUTING.md)

## Develop

Requires Go 1.24+ and Node 20+.

```sh
npm install            # workspaces: frontend + api-client
npm run generate       # regenerate TS client from openapi.yaml
npm run dev            # vite dev on :5173, proxies /api to :8080
```

In another shell:

```sh
cd apps/backend
DASH_DEV=1 go run ./cmd/dash   # API on :8080 (DASH_DEV skips embedded UI)
```

## Stack

Go + Gin + SQLite (single static binary, embeds the frontend) · React 18 +
Vite + TS strict + Tailwind + shadcn/ui + dnd-kit · OpenAPI-generated client.

## License

[MIT](LICENSE)
