FROM golang:1.26-alpine AS build

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

FROM alpine:3.22

RUN adduser -D -H -u 10001 app && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/dash-backend /app/dash-backend
COPY --from=build /src/db /app/db
RUN mkdir -p /data && chown -R app:app /data /app
USER app

ENV HTTP_ADDR=:8080
ENV MIGRATIONS_DIR=/app/db/migrations
EXPOSE 8080
CMD ["/app/dash-backend"]
