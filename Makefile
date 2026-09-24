VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
BIN     := bin/mailpeek

.PHONY: all build web packages run dev test test-race vet lint typecheck audit doctor bench e2e example check docker screenshots clean

all: build

## build: web UI + release binary with the UI embedded
build: web
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/mailpeek

node_modules: package.json package-lock.json
	npm ci
	@touch node_modules

web: node_modules
	npm run build -w @mailpeek/web

packages: node_modules
	npm run build -w @mailpeek/client -w @mailpeek/playwright

## run: build and start Mailpeek
run: build
	./$(BIN)

## dev: Go server on :8026 plus Vite with hot reload on :5173
dev:
	@echo "Run 'go run ./cmd/mailpeek' in one terminal and 'npm run dev -w @mailpeek/web' in another."

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

## lint: Go (gofmt, vet, golangci-lint) and TypeScript (eslint, prettier, UI size)
lint: node_modules
	@test -z "$$(gofmt -l . | tee /dev/stderr)" || (echo "gofmt needed on the files above" && exit 1)
	go vet ./...
	golangci-lint run ./...
	npm run lint
	npm run ui:size
	npm run ui:i18n

typecheck: packages
	npm run typecheck

## audit: known vulnerabilities in npm and Go dependencies
audit: node_modules
	npm run audit
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## doctor: React Doctor must report no issues (score 100)
doctor: node_modules
	npm run doctor

bench:
	go test -run '^$$' -bench . -benchmem ./internal/...

## e2e: Mailpeek's own Playwright suite against the release binary
e2e: build packages
	cd e2e && npx playwright test

## example: the documented Playwright example (demo app + Mailpeek binary)
example: build packages
	cd examples/playwright && MAILPEEK_BINARY="../../$(BIN)" npx playwright test

## check: every quality gate (lint, typecheck, build, tests, audit, react-doctor)
check: lint typecheck build test-race audit doctor
	npm test

## screenshots: regenerate docs/*.png (needs ports 1026/8026 free)
screenshots: build
	@./$(BIN) >/dev/null 2>&1 & pid=$$!; sleep 0.5; \
	node scripts/screenshots.mjs; status=$$?; kill $$pid; wait $$pid 2>/dev/null; [ $$status -eq 0 ] || exit $$status; \
	./$(BIN) --ui-auth admin:secret --smtp-auth app:secret >/dev/null 2>&1 & pid=$$!; sleep 0.5; \
	MAILPEEK_AUTH=1 node scripts/screenshots.mjs; status=$$?; kill $$pid; exit $$status

docker:
	docker build -t mailpeek --build-arg VERSION=$(VERSION) .

clean:
	rm -rf bin web/dist/assets web/dist/index.html web/dist/favicon.svg packages/*/dist
