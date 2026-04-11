# Cortex â€” development Makefile
# Usage: make <target>  [SVC=<service>]  [ARGS=<extra flags>]

SHELL := bash
.ONESHELL:
.DEFAULT_GOAL := help

# ---

BIN_DIR   := bin
# On Windows, Go requires .exe for cmd.exe compatibility
ifeq ($(OS),Windows_NT)
  CLI_BIN   := $(BIN_DIR)/cortex.exe
  RELAY_BIN := $(BIN_DIR)/relay.exe
else
  CLI_BIN   := $(BIN_DIR)/cortex
  RELAY_BIN := $(BIN_DIR)/relay
endif
SERVICES  := inference brain memory api courier
MODULES   := apps/cli apps/relay pkg/observe services/api services/arsenal services/atlas services/brain services/chat services/compass services/courier services/crucible services/forge services/inference services/memory services/nerva services/policy services/shell services/sovereign services/vault services/workspace

# ---

.PHONY: help
help:
	@echo ""
	@echo "  Cortex development commands"
	@echo ""
	@echo "  Stack"
	@echo "    make up              Start all services (detached)"
	@echo "    make down            Stop and remove containers"
	@echo "    make restart         Down then up"
	@echo "    make logs            Follow logs for all services"
	@echo "    make ps              Show container status"
	@echo ""
	@echo "  Build"
	@echo "    make build           Build all Docker images"
	@echo "    make rebuild SVC=x   Rebuild and restart one service (e.g. SVC=brain)"
	@echo "    make proto           Regenerate protobuf code for all services"
	@echo ""
	@echo "  CLI & Agents"
	@echo "    make cli             Build cortex CLI â†’ $(CLI_BIN)"
	@echo "    make relay           Build relay agent â†’ $(RELAY_BIN)"
	@echo "    make install         Install cortex CLI to ~/go/bin"
	@echo ""
	@echo "  Test & Quality"
	@echo "    make test            Run all Go tests across the workspace"
	@echo "    make tidy            Run go mod tidy across all modules"
	@echo "    make vet             Run go vet across the workspace"
	@echo ""
	@echo "  Quick access"
	@echo "    make chat ARGS='...' Run cortex chat via the CLI"
	@echo "    make health          Hit the API healthz endpoint"
	@echo ""

# ---

.PHONY: up
up:
	docker compose up -d

.PHONY: down
down:
	docker compose down

.PHONY: restart
restart: down up

.PHONY: logs
logs:
	docker compose logs -f $(SVC)

.PHONY: ps
ps:
	docker compose ps

# ---

.PHONY: build
build:
	docker compose build

.PHONY: rebuild
rebuild:
ifndef SVC
	$(error SVC is required. Usage: make rebuild SVC=brain)
endif
	docker compose build $(SVC)
	docker compose up -d --force-recreate $(SVC)
	@echo ""
	@echo "  $(SVC) rebuilt and restarted"

# ---

.PHONY: proto
proto:
	@for svc in brain inference memory courier atlas crucible vault chat arsenal compass sovereign workspace; do \
		echo "  generating proto for $$svc..."; \
		(cd services/$$svc && buf generate) || exit 1; \
	done
	@echo ""
	@echo "  proto generation complete"

# ---

.PHONY: cli
cli:
	@mkdir -p $(BIN_DIR)
	go build -o $(CLI_BIN) ./apps/cli/cmd/cortex
	@echo ""
	@echo "  built â†’ $(CLI_BIN)"
	@echo "  run:    ./$(CLI_BIN) chat \"your input here\""

.PHONY: relay
relay:
	@mkdir -p $(BIN_DIR)
	go build -o $(RELAY_BIN) ./apps/relay/cmd/relay
	@echo ""
	@echo "  built â†’ $(RELAY_BIN)"
	@echo "  run:    COURIER_ADDR=<host>:2020 ./$(RELAY_BIN)"

.PHONY: install
install:
	go install ./apps/cli/cmd/cortex
	@echo ""
	@echo "  cortex installed to $$(go env GOPATH)/bin/cortex"

# ---

.PHONY: test
test:
	@echo "  running tests across all modules..."
	@fail=0; \
	for mod in $(MODULES); do \
		echo ""; \
		echo "  â†’ $$mod"; \
		(cd $$mod && GOWORK=off go test ./... 2>&1) || fail=1; \
	done; \
	echo ""; \
	[ $$fail -eq 0 ] && echo "  all tests passed" || (echo "  some tests failed" && exit 1)

.PHONY: tidy
tidy:
	@echo "  tidying all modules..."
	@for mod in $(MODULES); do \
		echo "  â†’ $$mod"; \
		(cd $$mod && GOWORK=off go mod tidy) || exit 1; \
	done
	@echo ""
	@echo "  all modules tidied"

.PHONY: vet
vet:
	go vet ./...

# ---

.PHONY: chat
chat:
	@go run ./apps/cli/cmd/cortex chat $(ARGS)

.PHONY: health
health:
	@curl -s http://localhost:8000/healthz | python -m json.tool 2>/dev/null \
		|| curl -s http://localhost:8000/healthz

