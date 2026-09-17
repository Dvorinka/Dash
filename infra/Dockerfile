# syntax=docker/dockerfile:1

# ---- frontend ----
FROM node:24-alpine AS frontend
WORKDIR /src
COPY package.json package-lock.json openapi.yaml ./
COPY apps/frontend/package.json apps/frontend/package.json
COPY packages/api-client/package.json packages/api-client/package.json
RUN npm ci
COPY apps/frontend apps/frontend
COPY packages/api-client packages/api-client
RUN npm run build -w @dash/frontend

# ---- backend ----
FROM golang:1.27-alpine AS backend
WORKDIR /src
COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download
COPY apps/backend ./
COPY --from=frontend /src/apps/frontend/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/dash ./cmd/dash

# ---- final: static binary on distroless ----
# Runs as root so the ./data bind mount stays writable without PUID plumbing.
# Attack surface is a single static binary; no shell, no package manager.
FROM gcr.io/distroless/static-debian12
COPY --from=backend /out/dash /dash
ENV DASH_PORT=3000 DASH_DATA_DIR=/data
EXPOSE 3000
VOLUME ["/data"]
ENTRYPOINT ["/dash"]
