.PHONY: install dev up down lint typecheck test test-race test-integration test-e2e migrate-up migrate-down migrate-verify openapi-verify traceability-verify verify

install:
	cd backend && go mod download
	cd frontend && npm ci

dev:
	docker compose up --build

up:
	docker compose up -d --build

down:
	docker compose down

lint:
	cd backend && gofmt -l . | tee /tmp/simple-erp-gofmt && test ! -s /tmp/simple-erp-gofmt
	cd backend && go vet ./...
	cd frontend && npm run lint

typecheck:
	cd frontend && npm run typecheck

test:
	cd backend && go test ./...
	cd frontend && npm test

test-race:
	cd backend && go test -race ./...

test-integration:
	cd backend && go test -tags=integration ./test/integration/...

test-e2e:
	cd frontend && npm run test:e2e

migrate-up:
	cd backend && go run ./cmd/migrate up

migrate-down:
	cd backend && go run ./cmd/migrate down

migrate-verify:
	cd backend && go test -tags=integration ./test/integration -run Migration

openapi-verify:
	npx --yes @redocly/cli lint api/openapi.yaml

traceability-verify:
	./scripts/verify-traceability.sh

verify: lint typecheck test test-race openapi-verify traceability-verify
