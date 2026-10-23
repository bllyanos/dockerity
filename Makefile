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

DEV_MASTER_KEY := .data/master_key

# A throwaway master key for local runs. Never use it outside development.
$(DEV_MASTER_KEY):
	@mkdir -p .data
	@umask 077 && openssl rand -base64 32 > $@

run: build $(DEV_MASTER_KEY)
	DOCKERITY_DATA_DIR=./.data DOCKERITY_MASTER_KEY_FILE=./$(DEV_MASTER_KEY) DOCKERITY_LOG_LEVEL=debug ./$(BIN)

clean:
	rm -rf bin
