# ─────────────────────────────────────────────────────────────
#  Krypton — Makefile
#  Targets: build · run · test · lint · docker · migrate · clean
# ─────────────────────────────────────────────────────────────
SHELL := /bin/bash
.DEFAULT_GOAL := help

APP_NAME      := krypton
MODULE        := github.com/krypton-discord/krypton
BIN_DIR       := bin
BIN           := $(BIN_DIR)/$(APP_NAME)
MAIN_PKG      := ./cmd/bot
GO            := go
GOFLAGS       ?=
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT        ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS       := -s -w \
                 -X '$(MODULE)/internal/buildinfo.Version=$(VERSION)' \
                 -X '$(MODULE)/internal/buildinfo.Commit=$(COMMIT)' \
                 -X '$(MODULE)/internal/buildinfo.BuildDate=$(BUILD_DATE)'

# ── Colores ──────────────────────────────────────────────────
RESET  := \033[0m
BOLD   := \033[1m
GREEN  := \033[32m
YELLOW := \033[33m
CYAN   := \033[36m

.PHONY: help
help: ## Muestra esta ayuda
	@printf "$(BOLD)$(CYAN)Krypton — comandos disponibles$(RESET)\n\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-18s$(RESET) %s\n", $$1, $$2}'

# ═════════════════════════════════════════════════════════════
#  Build
# ═════════════════════════════════════════════════════════════
.PHONY: build
build: ## Compila el binario con ldflags de versión
	@mkdir -p $(BIN_DIR)
	@printf "$(CYAN)→ Compilando $(APP_NAME) $(VERSION)...$(RESET)\n"
	$(GO) build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) $(MAIN_PKG)
	@printf "$(GREEN)✓ Listo: $(BIN)$(RESET)\n"

.PHONY: build-linux
build-linux: ## Compila para Linux amd64 (deploy)
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BIN)-linux $(MAIN_PKG)

.PHONY: install
install: ## Instala el binario en $GOBIN
	$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" $(MAIN_PKG)

# ═════════════════════════════════════════════════════════════
#  Run
# ═════════════════════════════════════════════════════════════
.PHONY: run
run: ## Ejecuta el bot (carga .env)
	$(GO) run $(MAIN_PKG)

.PHONY: dev
dev: ## Hot reload con Air (si está instalado)
	@command -v air >/dev/null 2>&1 || { \
		printf "$(YELLOW)✗ air no instalado. Usa: go install github.com/air-verse/air@latest$(RESET)\n"; \
		exit 1; \
	}
	air -c .air.toml

# ═════════════════════════════════════════════════════════════
#  Testing
# ═════════════════════════════════════════════════════════════
.PHONY: test
test: ## Ejecuta todos los tests
	$(GO) test $(GOFLAGS) -race -count=1 ./...

.PHONY: test-cover
test-cover: ## Tests con reporte de cobertura
	$(GO) test $(GOFLAGS) -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@printf "$(GREEN)✓ Reporte: coverage.html$(RESET)\n"

.PHONY: bench
bench: ## Ejecuta benchmarks
	$(GO) test -bench=. -benchmem -run=^$$ ./...

# ═════════════════════════════════════════════════════════════
#  Calidad
# ═════════════════════════════════════════════════════════════
.PHONY: fmt
fmt: ## Formatea el código con gofmt + goimports
	$(GO) fmt ./...
	@command -v goimports >/dev/null 2>&1 && goimports -w -local $(MODULE) . || true

.PHONY: vet
vet: ## Ejecuta go vet
	$(GO) vet ./...

.PHONY: lint
lint: ## Ejecuta golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || { \
		printf "$(YELLOW)✗ golangci-lint no instalado.$(RESET)\n"; \
		exit 1; \
	}
	golangci-lint run --config .golangci.yml ./...

.PHONY: tidy
tidy: ## go mod tidy + verificación
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: check
check: fmt vet lint test ## Pipeline completo de calidad

# ═════════════════════════════════════════════════════════════
#  Base de datos
# ═════════════════════════════════════════════════════════════
.PHONY: migrate-up
migrate-up: ## Aplica migraciones SQL
	@command -v migrate >/dev/null 2>&1 || { \
		printf "$(YELLOW)✗ migrate no instalado.$(RESET)\n"; \
		exit 1; \
	}
	migrate -path ./internal/storage/migrations \
	        -database "$${POSTGRES_DSN}" up

.PHONY: migrate-down
migrate-down: ## Revierte la última migración
	migrate -path ./internal/storage/migrations \
	        -database "$${POSTGRES_DSN}" down 1

.PHONY: migrate-create
migrate-create: ## Crea migración. Uso: make migrate-create NAME=nombre
	@test -n "$(NAME)" || { printf "$(YELLOW)✗ Falta NAME=$(RESET)\n"; exit 1; }
	migrate create -ext sql -dir ./internal/storage/migrations -seq $(NAME)

# ═════════════════════════════════════════════════════════════
#  Docker
# ═════════════════════════════════════════════════════════════
.PHONY: docker-build
docker-build: ## Construye imagen Docker
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t krypton:$(VERSION) \
		-f deployments/docker/Dockerfile .

.PHONY: docker-up
docker-up: ## Levanta stack local (postgres + redis + bot)
	docker compose up -d --build

.PHONY: docker-down
docker-down: ## Baja el stack local
	docker compose down

.PHONY: docker-logs
docker-logs: ## Muestra logs del bot
	docker compose logs -f bot

# ═════════════════════════════════════════════════════════════
#  Limpieza
# ═════════════════════════════════════════════════════════════
.PHONY: clean
clean: ## Limpia artefactos
	rm -rf $(BIN_DIR) coverage.out coverage.html dist/ tmp/
	$(GO) clean -cache -testcache

.PHONY: nuke
nuke: clean ## Limpieza agresiva (¡ojo con vendor/!)
	rm -rf vendor/

# ═════════════════════════════════════════════════════════════
#  Info
# ═════════════════════════════════════════════════════════════
.PHONY: version
version: ## Muestra la versión calculada
	@printf "Version:    $(VERSION)\n"
	@printf "Commit:     $(COMMIT)\n"
	@printf "Build date: $(BUILD_DATE)\n"