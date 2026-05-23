APP_NAME := danqing-teams
SERVER_BIN := $(CURDIR)/out/server/$(APP_NAME)
FRONTEND_DIR := $(CURDIR)/frontend
OUT_DIR := $(CURDIR)/out
FRONTEND_DIST := $(OUT_DIR)/frontend/dist
RELEASE_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

export DQ_APP_NAME := $(APP_NAME)

.PHONY: help dev start stop \
	frontend-install frontend-dev frontend-build frontend-typecheck \
	test test-integration \
	build-server build-all clean \
	pack-prereqs pack-macos-desktop pack-linux-server pack-windows-desktop

help:
	@echo "DanQing Teams"
	@echo ""
	@echo "Dev:       dev | start | stop"
	@echo "Frontend:  frontend-install | frontend-dev | frontend-build | frontend-typecheck"
	@echo "Test:      test | test-integration"
	@echo "Build:     build-server | build-all | clean"
	@echo "Release:   pack-macos-desktop | pack-linux-server | pack-windows-desktop"

dev start:
	@chmod +x scripts/*.sh
	@./scripts/start.sh

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

test:
	go test ./... -count=1

test-integration:
	go test ./test/integration/... -count=1

build-server:
	@mkdir -p $(OUT_DIR)/server
	go build -o $(SERVER_BIN) ./cmd/server

build-all: frontend-build build-server

clean:
	rm -rf $(OUT_DIR)

pack-prereqs:
	@command -v npm >/dev/null 2>&1 || (echo "npm not found" >&2; exit 1)
	@command -v cargo >/dev/null 2>&1 || (echo "cargo not found" >&2; exit 1)
	@echo "pack-prereqs OK"

pack-macos-desktop: pack-prereqs frontend-build
	@chmod +x scripts/*.sh
	@./scripts/pack_desktop_macos.sh

pack-linux-server: frontend-build build-server
	@chmod +x scripts/*.sh
	@RELEASE_VERSION=$(RELEASE_VERSION) ./scripts/pack_linux_server.sh

pack-windows-desktop: pack-prereqs frontend-build
	@chmod +x scripts/*.sh
	@./scripts/pack_desktop_windows.sh
