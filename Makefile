include .env
export

# Metadatos de compilación: los publica GET /api/v1/version.
SOLV_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
SOLV_COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
SOLV_BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: run
run:
	@echo "=>  Levantando backend en modo desarrollo..."
	@cd backend && go run ./cmd/api

.PHONY: build
build:
	@echo "=>  Compilando el binario de la API..."
	@cd backend && go build -o bin/api ./cmd/api

.PHONY: dev
dev:
	@./dev.sh

.PHONY: lint\:backend
lint\:backend:
	@echo "=>  golangci-lint (backend)..."
	@cd backend && PATH="$$(go env GOPATH)/bin:$$PATH" golangci-lint run ./...

.PHONY: images
images:
	@echo "=>  Construyendo solv-backend:$(SOLV_VERSION) y solv-frontend:$(SOLV_VERSION)..."
	@SOLV_VERSION=$(SOLV_VERSION) SOLV_COMMIT=$(SOLV_COMMIT) SOLV_BUILD_DATE=$(SOLV_BUILD_DATE) \
		docker compose build

.PHONY: version
version:
	@curl -fsS http://127.0.0.1:3000/api/v1/version || echo "El backend no responde en 127.0.0.1:3000"

.PHONY: upgrade
upgrade:
	@[ -n "$(VERSION)" ] || { echo "uso: make upgrade VERSION=2.1.0"; exit 1; }
	@./scripts/upgrade.sh --version "$(VERSION)"

.PHONY: token
token:
	@./scripts/gen-token.sh

.PHONY: test-impact-plan
test-impact-plan:
	@node tools/test-impact/cli.mjs plan --base $${BASE:-origin/main} --working-tree --format text

.PHONY: test-impact
test-impact:
	@node tools/test-impact/run.mjs --all --base $${BASE:-origin/main} --working-tree --report impact-report.json
	@node tools/test-impact/cli.mjs report --file impact-report.json

.PHONY: test-impact-suite
test-impact-suite:
	@node tools/test-impact/run.mjs --partial --suite $${SUITE:-build} --if-selected \
		--base $${BASE:-origin/main} --working-tree --report impact-report-$${SUITE:-build}.json

.PHONY: test-selector
test-selector:
	@node tools/test-impact/cli.mjs validate
	@node --test tools/test-impact/test/*.test.mjs


