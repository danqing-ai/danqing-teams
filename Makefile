APP_NAME := danqing-teams
SERVER_BIN := $(CURDIR)/out/server/$(APP_NAME)
CLI_BIN := $(CURDIR)/out/server/$(APP_NAME)-cli
TUI_BIN := $(CURDIR)/out/server/$(APP_NAME)-tui
FRONTEND_DIR := $(CURDIR)/frontend
OUT_DIR := $(CURDIR)/out
FRONTEND_DIST := $(OUT_DIR)/frontend/dist
RELEASE_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -w -X 'danqing-teams/server/api/v1.Version=$(RELEASE_VERSION)'

export DQ_APP_NAME := $(APP_NAME)

.PHONY: help backend dev-web dev-desktop dev-cli dev-tui stop \
	frontend-install frontend-dev frontend-build frontend-typecheck \
	check-layers test test-integration \
	build-go build-server build-cli build-tui build-sidecar build build-all clean \
	pack-prereqs pack-macos-desktop pack-linux-server pack-windows-desktop

help:
	@echo "DanQing Teams"
	@echo ""
	@echo "Dev:"
	@echo "  backend       Start backend only (:7801) — for Go debugger"
	@echo "  dev-web       Backend + Vite (browser :5801)"
	@echo "  dev-desktop   Backend + Tauri webview (set SKIP_BACKEND=1 to use external backend)"
	@echo "  dev-cli       Run CLI directly (no server needed)"
	@echo "  dev-tui       Run TUI directly (no server needed)"
	@echo "  stop          Stop all dev processes"
	@echo ""
	@echo "Frontend:  frontend-install | frontend-dev | frontend-build | frontend-typecheck"
	@echo "Test:      check-layers | test | test-integration"
	@echo "Build:     build | build-all | build-go | build-server | build-cli | build-tui | build-sidecar | clean"
	@echo "Release:   pack-macos-desktop | pack-linux-server | pack-windows-desktop"

# Backend only (for Go debugger or separate frontend)
backend:
	@chmod +x scripts/*.sh
	@./scripts/start_backend.sh

# Web browser development
dev-web:
	@chmod +x scripts/*.sh
	@./scripts/start_web.sh

# Desktop (Tauri) development
dev-desktop:
	@chmod +x scripts/*.sh
	@./scripts/start_desktop.sh

# CLI (no server needed)
dev-cli:
	go run ./cli

# TUI (no server needed)
dev-tui:
	go run ./tui

stop:
	@chmod +x scripts/*.sh
	@./scripts/stop.sh

frontend-install:
	cd $(FRONTEND_DIR) && npm install

frontend-dev: frontend-install
	cd $(FRONTEND_DIR) && npm run dev

frontend-build: frontend-install
	cd $(FRONTEND_DIR) && npm run build

frontend-typecheck: frontend-install
	cd $(FRONTEND_DIR) && npm run typecheck

check-layers:
	go run scripts/check_layers.go

test: check-layers
	go test ./... -count=1

test-integration:
	go test ./test/integration/... -count=1

build-go: build-server build-cli build-tui

build-server:
	@mkdir -p $(OUT_DIR)/server
	go build -ldflags "$(LDFLAGS)" -o $(SERVER_BIN) ./server

build-cli:
	@mkdir -p $(OUT_DIR)/server
	go build -ldflags "$(LDFLAGS)" -o $(CLI_BIN) ./cli

build-tui:
	@mkdir -p $(OUT_DIR)/server
	go build -ldflags "$(LDFLAGS)" -o $(TUI_BIN) ./tui

build-sidecar:
	@chmod +x scripts/*.sh
	@./scripts/build_sidecar.sh

build build-all: frontend-build build-go

clean:
	rm -rf $(OUT_DIR)

pack-prereqs:
	@command -v npm >/dev/null 2>&1 || (echo "npm not found" >&2; exit 1)
	@command -v cargo >/dev/null 2>&1 || (echo "cargo not found" >&2; exit 1)
	@echo "pack-prereqs OK"

pack-macos-desktop: pack-prereqs frontend-build
	@chmod +x scripts/*.sh
	@RELEASE_VERSION=$(RELEASE_VERSION) ./scripts/pack_desktop_macos.sh

pack-linux-server: frontend-build build-go
	@chmod +x scripts/*.sh
	@RELEASE_VERSION=$(RELEASE_VERSION) ./scripts/pack_linux_server.sh

pack-windows-desktop: pack-prereqs frontend-build
	@chmod +x scripts/*.sh
	@RELEASE_VERSION=$(RELEASE_VERSION) ./scripts/pack_desktop_windows.sh
