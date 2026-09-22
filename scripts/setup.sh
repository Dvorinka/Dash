#!/bin/sh
# Dash setup from a clone — builds the image locally and starts the stack.
# For the prebuilt image path, use ../install.sh instead.
set -eu

cd "$(dirname "$0")/.."

command -v docker >/dev/null 2>&1 || { echo "error: docker not found" >&2; exit 1; }
docker compose version >/dev/null 2>&1 || { echo "error: docker compose plugin not found" >&2; exit 1; }

docker compose up -d --build

echo ""
echo "Dash is up: http://localhost:3000"
echo "Logs: docker compose logs -f"
echo ""
echo "For development instead:"
echo "  npm install && npm run dev          # vite on :5173"
echo "  cd apps/backend && DASH_DEV=1 go run ./cmd/dash   # API on :8080"
