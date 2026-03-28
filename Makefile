.PHONY: all wasm wasm-exec kill-port server dev build lint sec test ci docker-build deploy deploy-agent logs clean

WASM_OUT = server/web/game.wasm
BINARY   = purr
URL      = http://localhost:8071
PORT     ?= 8071
DOCKER_IMAGE ?= purr:latest
GOLANGCI_LINT_VERSION ?= v2.8.0
GOSEC_VERSION ?= v2.22.2

# ── deployment ────────────────────────────────────────────────────────────────
REMOTE_HOST ?= daemon
REMOTE_DIR  ?= ~/cats
REMOTE_BIN  := $(REMOTE_DIR)/$(BINARY)
REMOTE_LOG  := $(REMOTE_DIR)/server.log

ifeq ($(shell uname -s),Darwin)
BROWSER_OPEN = open
else
BROWSER_OPEN = xdg-open
endif

all: lint sec wasm kill-port
	@sleep 3 && $(BROWSER_OPEN) $(URL) >/dev/null 2>&1 &
	go run ./server/

wasm: lint sec
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o $(WASM_OUT) ./game/

wasm-exec:
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" server/web/wasm_exec.js

kill-port:
	@lsof -ti :8071 | xargs kill -9 2>/dev/null || true

server: lint sec wasm kill-port
	@sleep 3 && $(BROWSER_OPEN) $(URL) >/dev/null 2>&1 &
	go run ./server/

dev: lint sec wasm kill-port
	@sleep 3 && $(BROWSER_OPEN) $(URL) >/dev/null 2>&1 &
	go run ./server/

build: lint sec wasm
	go build -ldflags="-s -w" -o $(BINARY) ./server/

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

sec:
	go run github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION) ./...

test:
	go test ./...

ci: lint sec test wasm build

docker-build: lint sec
	docker build -t $(DOCKER_IMAGE) .

deploy: lint sec wasm
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY) ./server/
	ssh $(REMOTE_HOST) "mkdir -p $(REMOTE_DIR) && pkill -x $(BINARY) || true; sleep 1; rm -f $(REMOTE_BIN)"
	scp $(BINARY) $(REMOTE_HOST):$(REMOTE_BIN)
	scp config.yaml $(REMOTE_HOST):$(REMOTE_DIR)/config.yaml
	scp scripts/start-server.sh $(REMOTE_HOST):$(REMOTE_DIR)/start-server.sh
	ssh $(REMOTE_HOST) "chmod +x $(REMOTE_DIR)/start-server.sh && PORT=$(PORT) $(REMOTE_DIR)/start-server.sh"

deploy-agent: lint sec wasm
	DEPLOY_DIR="$${DEPLOY_DIR:-$$HOME/cats}"; \
	mkdir -p "$$DEPLOY_DIR/data" && \
	docker build -t $(DOCKER_IMAGE) . && \
	cp compose.yaml "$$DEPLOY_DIR/compose.yaml" && \
	cp config.yaml "$$DEPLOY_DIR/config.yaml" && \
	IMAGE_NAME=$(DOCKER_IMAGE) PORT=$(PORT) docker compose -f "$$DEPLOY_DIR/compose.yaml" up -d --force-recreate

logs:
	ssh $(REMOTE_HOST) "tail -f $(REMOTE_LOG)"

clean:
	rm -f $(WASM_OUT) $(BINARY) server/web/wasm_exec.js
