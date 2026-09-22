# Installation

Dash ships as a single container image or a single static binary. All state
lives in one directory: a SQLite database and uploaded icons.

## One-line install

```sh
curl -fsSL https://raw.githubusercontent.com/Dvorinka/Dash/main/install.sh | sh
```

Creates `./dash`, pulls the image, and starts on port 3000. Override with
`DASH_DIR` / `DASH_PORT` environment variables.

## Docker (manual)

```sh
docker run -d --name dash \
  -v ./data:/data \
  -p 3000:3000 \
  --restart unless-stopped \
  ghcr.io/dvorinka/dash:latest
```

Or compose:

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

Open http://localhost:3000.

To build the image locally instead of pulling:

```sh
docker compose up --build
```

## Binary

Release archives contain a static `dash` binary (no CGO, no dependencies).

```sh
tar xzf dash_linux_amd64.tar.gz
./dash   # listens on :8080, stores data in ./data
```

Set `DASH_PORT` and `DASH_DATA_DIR` to change the defaults.

## dash-agent (system monitoring)

The agent collects CPU, memory, disk, network, load, temperatures, and Docker
container state, then pushes a JSON sample to Dash every few seconds.

1. In the UI, go to **Systems → New system**, copy the generated token.
2. Download the `dash-agent` binary for the host's architecture from the
   release page, or build it:

   ```sh
   cd apps/backend && go build -o dash-agent ./cmd/dash-agent
   ```

3. Run it:

   ```sh
   dash-agent -url http://dash:3000 -token <token>
   ```

### Flags / environment

| Flag | Env | Default | Purpose |
|---|---|---|---|
| `-url` | `DASH_URL` | — | Dash base URL |
| `-token` | `DASH_TOKEN` | — | system token from the UI |
| `-interval` | `DASH_INTERVAL` | `10` | push interval, seconds |
| `-once` | — | `false` | print one sample and exit (no push) |

If `/var/run/docker.sock` exists, container states are included automatically.

### systemd

`packaging/dash-agent.service` is a ready-made unit:

```sh
sudo install -m755 dash-agent /usr/local/bin/dash-agent
sudo install -m644 packaging/dash-agent.service /etc/systemd/system/
sudo mkdir -p /etc/dash
printf 'DASH_URL=http://dash:3000\nDASH_TOKEN=<token>\n' | sudo tee /etc/dash/agent.env
sudo systemctl enable --now dash-agent
```

### Docker (limited)

`infra/Dockerfile.agent` builds a minimal agent image. A containerized agent
only sees its own `/proc` — mount the host's procfs and use host PID mode for
meaningful numbers:

```sh
docker run -d --name dash-agent --pid=host \
  -v /proc:/host/proc:ro \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -e DASH_URL=http://dash:3000 -e DASH_TOKEN=<token> \
  ghcr.io/dvorinka/dash-agent:latest
```

The systemd install above is the recommended path.

## Reverse proxy

Dash serves the UI and API on one port — a plain reverse proxy works:

```nginx
location / {
    proxy_pass http://127.0.0.1:3000;
    proxy_set_header Host $host;
}
```

## Upgrading

Pull the new image and restart, or replace the binary. Goose migrations run
automatically on boot; `data/` carries forward.

```sh
docker pull ghcr.io/dvorinka/dash:latest && docker compose up -d
```
