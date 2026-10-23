BIN := bin/dockerity
GOLANGCI_LINT ?= golangci-lint

.PHONY: build build-go web web-deps test lint lint-go lint-web run clean

build: web build-go

build-go:
	CGO_ENABLED=0 go build -o $(BIN) ./cmd/dockerity

web-deps:
	cd web && npm ci

web:
	cd web && npm run build

test:
	go test ./...

lint: lint-go lint-web

lint-go:
	$(GOLANGCI_LINT) run ./...

lint-web:
	cd web && npm run lint

run: build
	DOCKERITY_DATA_DIR=./.data DOCKERITY_LOG_LEVEL=debug ./$(BIN)

clean:
	rm -rf bin
