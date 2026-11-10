include .env
export

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


