# Monitoring your infrastructure

Dash displays and alerts; it does not collect metrics beyond what
`dash-agent` pushes. For real infrastructure monitoring, run the reference
stack in [`infra/monitoring/`](../infra/monitoring/) next to Dash — it works
on any host. A small Debian LXC on the Proxmox box is a good home.

## What watches what

| Layer | Tool | Covers |
|---|---|---|
| Physical hosts | Beszel hub + agents | Proxmox host, Windows 11 box — CPU/mem/disk/net/temps |
| Cluster + apps | Prometheus + pve-exporter | Every LXC/VM via the PVE API (no guest agents), Dash's own `/api/metrics` |
| Dashboards | Grafana | Graphs over Prometheus, embedded into Dash via iframes |
| Up/down + alerts | Dash monitors | Router, hosts, services — with notification transports and incidents |

Two agents total — Beszel deliberately owns only the physical machines; the
PVE API already knows everything about the guests.

## 1. Deploy the stack

```sh
cd infra/monitoring
cp pve.yml.example pve.yml      # fill in the PVE API token (see below)
# edit prometheus.yml: replace PVE_HOST and DASH_HOST with real addresses
docker compose up -d
```

UIs: Prometheus `:9090`, Grafana `:3001`, Beszel `:8090`, pve-exporter `:9221`.

### Proxmox API token (for pve-exporter)

On the PVE host:

```sh
pveum user add dashmon@pve --comment "read-only monitoring"
pveum aclmod / -user dashmon@pve -role PVEAuditor
```

Then **Datacenter → Permissions → API Tokens → Add** a token named `dash`
for `dashmon@pve`, uncheck *Privilege Separation*, copy the secret into
`pve.yml`. Token auth means no password in the file.

### Beszel agents (physical hosts)

1. Open Beszel `http://<hub>:8090`, create the admin account.
2. **Add System** → the hub generates the exact install command with the
   key embedded — binary/systemd for the PVE host, Windows build for the
   dev box (run it under NSSM or Task Scheduler for boot persistence).
3. Agents listen on `:45876`; the hub connects to them. On the same LAN
   this needs no extra plumbing.

## 2. Show it in Dash — no code required

### Live values → JSON value widget

Add item → widget type **JSON value**:

**Beszel** — fields are version-dependent; browse the API to pick a path:

- URL: `http://<beszel>:8090/api/collections/systems/records`
- Header: `Authorization: Bearer <token>` — mint a superuser token/key in
  the PocketBase admin at `http://<beszel>:8090/_/`
- Path: e.g. `items[0].info.cpu` (verify in the JSON response)

**Prometheus** — instant queries return JSON:

- URL: `http://<prometheus>:9090/api/v1/query?query=pve_cpu_usage_ratio`
- Path: `data.result[0].value[1]`
- Useful queries: `pve_cpu_usage_ratio`, `pve_memory_usage_ratio`,
  `pve_disk_usage_ratio`, `pve_up`

### Graphs → Embed widget

Grafana dashboard → share a panel → **Embed** gives a `d-solo` URL. Paste it
into a Dash **Embed** widget (append `&kiosk` for chrome-free panels). The
shipped compose enables anonymous Viewer + embedding, so iframes render
without login on the LAN — remove the `GF_AUTH_ANONYMOUS_*` and
`GF_SECURITY_ALLOW_EMBEDDING` env lines if you'd rather require sign-in, and
use Grafana *public dashboard* links instead.

The whole Beszel UI also iframes cleanly if you want its charts in a tile.

## 3. Alert on outages → Dash monitors

The point of the exercise: when the unmanaged switch hangs, everything
*behind* it drops at once while router-side checks stay green. That
co-failure pattern — timestamped, pushed to your phone — is as close to
"the switch died" as an unmanaged box allows.

Recommended monitors:

| Target | Type | Why |
|---|---|---|
| Router / gateway IP | Ping | Upstream reference point |
| Proxmox host | Ping + HTTP `:8006` | Behind the switch |
| Each LXC's service | HTTP/TCP | Per-app liveness |
| Win11 box | TCP `:3389` (RDP) | Behind the switch |
| Prometheus/Grafana/Beszel | HTTP | Watch the watchers |

Settings → Notifications: configure ntfy/Telegram/etc. once and every
down-transition alerts you with an incident record.

**Possible switch fix, unproven:** cheap-switch hangs are often EEE
(energy-efficient ethernet) negotiation bugs. `ethtool --set-eee <nic> eee
off` on the host NICs is a reasonable mitigation. Failing that, the usual
suspects are heat and the power brick.

## 4. Files

```
infra/monitoring/compose.yaml            stack definition
infra/monitoring/prometheus.yml          scrape config (edit hostnames)
infra/monitoring/pve.yml.example         PVE token template → copy to pve.yml
infra/monitoring/grafana/datasources.yaml  Prometheus source, auto-provisioned
```

`pve.yml` holds a credential and is gitignored. Stack design rationale:
[`docs/specs/2026-10-03-external-monitoring.md`](specs/2026-10-03-external-monitoring.md).
