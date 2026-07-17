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
