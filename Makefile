BACKEND_DIR := backend
FRONTEND_DIR := frontend
COMPOSE_FILE := docker-compose.yml
DATABASE_URL := postgres://postgres:postgres@localhost:5432/darts_league?sslmode=disable

.PHONY: test test-backend test-frontend test-e2e build-frontend dev-backend dev-backend-db dev-frontend db-up db-down db-logs
.PHONY: up down logs logs-backend logs-frontend logs-db rebuild

test: test-backend test-frontend

test-backend:
	cd $(BACKEND_DIR) && go test ./...

test-frontend:
	cd $(FRONTEND_DIR) && npm test

test-e2e:
	cd $(FRONTEND_DIR) && npm run test:e2e

build-frontend:
	cd $(FRONTEND_DIR) && npm run build

dev-backend:
	cd $(BACKEND_DIR) && go run ./cmd/api

dev-backend-db:
	cd $(BACKEND_DIR) && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/api

dev-frontend:
	cd $(FRONTEND_DIR) && npm run dev

db-up:
	docker compose -f $(COMPOSE_FILE) up -d postgres

up:
	docker compose -f $(COMPOSE_FILE) up -d --build

rebuild:
	docker compose -f $(COMPOSE_FILE) up -d --build --force-recreate

db-down:
	docker compose -f $(COMPOSE_FILE) down

down:
	docker compose -f $(COMPOSE_FILE) down

db-logs:
	docker compose -f $(COMPOSE_FILE) logs -f postgres

logs:
	docker compose -f $(COMPOSE_FILE) logs -f

logs-backend:
	docker compose -f $(COMPOSE_FILE) logs -f backend

logs-frontend:
	docker compose -f $(COMPOSE_FILE) logs -f frontend

logs-db:
	docker compose -f $(COMPOSE_FILE) logs -f postgres
