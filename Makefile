.PHONY: check check-go check-web check-reference check-storage fmt-check build dev-api dev-web db-up db-down db-migrate

check: check-go check-web check-reference

fmt-check:
	@files="$$(gofmt -l cmd internal)"; if [ -n "$$files" ]; then printf 'Run gofmt on:\n%s\n' "$$files"; exit 1; fi

check-go: fmt-check
	go mod verify
	go vet ./...
	go test -race ./...
	go build -o bin/confignostic-api ./cmd/confignostic-api
	go build -o bin/confignostic-db ./cmd/confignostic-db

check-storage:
	@test -n "$$CONFIGNOSTIC_TEST_DATABASE_URL" || (echo 'Set CONFIGNOSTIC_TEST_DATABASE_URL to a disposable PostgreSQL server'; exit 1)
	@test -n "$$CONFIGNOSTIC_TEST_PG_TOOLS" || (echo 'Set CONFIGNOSTIC_TEST_PG_TOOLS to the PostgreSQL tool wrapper'; exit 1)
	go test -race -tags=integration -count=1 -timeout=5m -v ./internal/storage

check-web:
	npm --prefix web ci
	npm --prefix web run check
	npm --prefix web run build

check-reference:
	node --test testdata/handoff/reference.test.mjs

build:
	go build -o bin/confignostic-api ./cmd/confignostic-api
	go build -o bin/confignostic-db ./cmd/confignostic-db
	npm --prefix web run build

dev-api:
	go run ./cmd/confignostic-api

dev-web:
	npm --prefix web run dev

db-up:
	docker compose -f compose.dev.yaml up -d --wait --wait-timeout 60

db-down:
	docker compose -f compose.dev.yaml down

db-migrate:
	go run ./cmd/confignostic-db migrate
