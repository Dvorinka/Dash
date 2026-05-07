# Build stage for frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json* ./frontend/
WORKDIR /app/frontend
RUN npm ci
WORKDIR /app
COPY frontend ./frontend
WORKDIR /app/frontend
RUN npm run build

# Build stage for backend
FROM golang:1.26-alpine AS backend-builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY backend/go.mod backend/go.sum* ./backend/
WORKDIR /src/backend
RUN go mod download
WORKDIR /src
COPY backend ./backend
COPY db ./db
WORKDIR /src/backend
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/dash-backend ./cmd/server

# Final stage with supervisord
FROM alpine:3.22

# Install supervisord
RUN apk add --no-cache ca-certificates supervisor

# Create users
RUN adduser -D -H -u 10001 app && \
    addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# Setup directories
RUN mkdir -p /app/frontend /data /var/log/supervisor && \
    chown -R app:app /data /app && \
    chown -R nextjs:nodejs /app/frontend

# Copy backend
COPY --from=backend-builder /out/dash-backend /app/dash-backend
COPY --from=backend-builder /src/db /app/db

# Copy frontend
COPY --from=frontend-builder /app/frontend/public /app/frontend/public
COPY --from=frontend-builder --chown=nextjs:nodejs /app/frontend/.next/standalone /app/frontend/
COPY --from=frontend-builder --chown=nextjs:nodejs /app/frontend/.next/static /app/frontend/.next/static

# Copy supervisord config
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Expose ports
EXPOSE 8080 3000

# Start supervisord
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
