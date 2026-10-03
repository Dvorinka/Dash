# External Monitoring Stack — Design

Date: 2026-10-03
Status: approved in chat 2026-10-03 — implementing

## 1. Goal

Monitor a homelab (one Proxmox host with LXCs, one always-on Windows 11 dev
box, an unmanaged Ugreen CM834 switch, a router) with the collection layer
living **outside** Dash. Dash consumes data for display and owns up/down
alerting. Trigger: an unmanaged switch hung overnight with no way to know
when or what failed.

## 2. Lane split — why two engines

Two engines were requested. They earn their place only if lanes don't
overlap:

| Tool | Lane | Scope |
|---|---|---|
| Beszel (hub + agents) | The two **physical** machines: PVE host, Win11 | Agent push, Windows support, instant host tiles, standalone alerts |
| Prometheus + `prometheus-pve-exporter` | Everything **scrapable**: all LXCs/VMs via the PVE API (no guest agents), Dash's own `/api/metrics` | System of record, PromQL |
| Grafana | Graphing front for Prometheus | Panel iframes for Dash |

Deliberately absent: `node_exporter`, `windows_exporter` — Beszel already
owns the physical hosts. Two agents total, not two per machine.

## 3. Dash integration — zero code changes

Existing primitives cover the whole display layer:

- **JSON-path widget** → Beszel PocketBase API and Prometheus
  `/api/v1/query` (`data.result[0].value[1]` shape already walkable;
  `header` config field carries `Authorization: Bearer …`)
- **Embed widget** → Grafana panel URLs (anonymous Viewer + embedding
  enabled on the LAN) and the Beszel UI
- **Monitors** → ping/TCP/HTTP checks + notification transports +
  auto-incidents = the "something died at 03:12" alarm the user lacked
- **Service links** → per-console deep links

## 4. The switch problem

The Ugreen CM834 is unmanaged: no IP, no SNMP, no telemetry. Root cause is
unobservable. What monitoring delivers instead:

- Monitors on router IP, PVE host, and one service per LXC — all reached
  *through* the switch. When it hangs, behind-switch targets fail together
  while router-side checks stay green: a timestamped "it was the switch"
  signature plus an alert to ntfy/Telegram.
- Plausible mitigation (not proven): disable EEE on host NICs —
  `ethtool --set-eee <nic> eee off`. Cheap-switch EEE hangs are a known
  failure mode.

## 5. Deliverables

```
infra/monitoring/
  compose.yaml            # prometheus + grafana + pve-exporter + beszel-hub
  prometheus.yml          # scrape config: pve (via exporter) + dash /api/metrics
  pve.yml.example         # PVE API token config (copy → pve.yml, gitignored)
  grafana/datasources.yaml  # auto-provisions the Prometheus source
docs/monitoring.md        # deploy guide + Dash widget/monitor recipes
```

No application code. No schema changes. Stack runs anywhere — an LXC on the
Proxmox host is the suggested home.

## 6. Security notes

- Grafana anonymous **Viewer** + `allow_embedding` are set for LAN iframe
  use; both are called out in the guide and removable.
- `pve.yml` carries an API token — `pve.yml.example` is committed, the real
  file is documented as gitignored-adjacent (lives next to the compose,
  outside the repo if preferred).
- Widget secrets already live in item config JSON per the original spec.

## 7. Verification

- `docker compose -f infra/monitoring/compose.yaml config` valid
- `promtool check config prometheus.yml` (dockerized promtool) valid
- Docs-only otherwise; repo gate (typecheck/lint/build/go tests) unchanged
