.PHONY: test run dev build-ui build-all

test:
	go test ./... -count=1

run:
	go run ./cmd/server

dev:
	TEAMS_AUTO_APPROVE=false go run ./cmd/server

build-ui:
	cd frontend && npm ci && npm run build

build-all: build-ui
	go build -o bin/danqing-teams ./cmd/server
