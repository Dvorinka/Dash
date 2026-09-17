# Dash

A self-hosted homelab dashboard. CasaOS-grade ease of use, Vercel-grade design.
Single container, UI-managed, no YAML, no auth.

**Status:** pre-release. See [ROADMAP.md](ROADMAP.md).

## Run

```sh
docker run -v ./data:/data -p 3000:3000 dash
```

or

```sh
docker compose up
```

Then open http://localhost:3000.

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
