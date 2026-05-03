BACKEND_DIR := backend
DATABASE_URL ?= postgres://dash:dash@localhost:5432/dash?sslmode=disable
MIGRATIONS_DIR := db/migrations

.PHONY: dev dev-down dev-logs backend-dev backend-test db-migrate-up db-migrate-down openapi-validate sqlc docker-up

# Start entire app in Docker (postgres + backend + frontend)
dev:
	docker compose up --build -d
	@echo "App starting..."
	@echo "Frontend: http://localhost:3000"
	@echo "Backend API: http://localhost:8080"
	@echo "Run 'make dev-logs' to see logs"

# Stop the Docker app
dev-down:
	docker compose down

# View logs from all services
dev-logs:
	docker compose logs -f

# Legacy docker-up (same as dev)
docker-up:
	docker compose up --build

# Local backend development (requires local postgres)
backend-dev:
	cd $(BACKEND_DIR) && APP_ENV=development DATABASE_URL="$(DATABASE_URL)" DATA_DIR=../data go run ./cmd/server

backend-test:
	cd $(BACKEND_DIR) && go test ./...

db-migrate-up:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

db-migrate-down:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

openapi-validate:
	npx --yes @redocly/cli@latest lint openapi/openapi.yaml

sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
