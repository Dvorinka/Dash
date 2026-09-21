# v1.0.0 launch post — draft

For r/selfhosted, HN Show, and similar. Edit freely before posting.

---

**Title:** Dash — a self-hosted homelab dashboard with monitoring built in.
Single Go binary, no YAML, no auth.

**Body:**

I built Dash because my homelab had outgrown a bookmark page: I wanted service
links, uptime checks, domain expiry, server stats, and a status page — without
running five separate tools or maintaining a YAML file.

What it does:

- **Board** — drag-and-drop service grid, multiple layouts, dark/light, icon
  upload. Imports your existing Homepage (`services.yml`), Homarr, or Dashy
  config in one click.
- **Monitors** — HTTP/TCP/ping/DNS/keyword/JSON-path checks plus push monitors
  for cron jobs. Webhook alerts (Slack/Discord/ntfy). Uptime history.
- **Domains** — WHOIS/RDAP, DNS, TLS certificate expiry, provider detection.
  Inspired by Domain Locker, minus the paid APIs.
- **Systems** — a ~4 MB static agent pushes CPU/mem/disk/net/temps/Docker
  container states. Inspired by Beszel, push-JSON instead of SSH.
- **Ops** — incidents, maintenance windows, public status pages, SVG badges,
  Prometheus `/api/metrics`.

Deployment is one container:

```sh
docker run -d -v ./data:/data -p 3000:3000 ghcr.io/dvorinka/dash:latest
```

SQLite inside, no database to babysit, no accounts. Deliberately no auth —
it expects a trusted LAN or your own proxy in front.

Stack: Go + Gin + SQLite (single static binary serving an embedded React UI).
MIT licensed.

Repo: https://github.com/Dvorinka/Dash
Screenshots are in the README.

Happy to answer questions — feedback welcome, especially on which monitor
types or integrations to add next.

---

**HN variant (Show HN guidelines — terse):**

Show HN: Dash – self-hosted homelab dashboard with monitoring, one Go binary

Dash combines a service board, uptime monitors, domain/SSL watching, system
metrics via a tiny push agent, and public status pages. Single container or
single binary, SQLite storage, no YAML, no auth (LAN-trust model). MIT.
https://github.com/Dvorinka/Dash
