BIN := bin/dockerity

.PHONY: build test lint run clean

build:
	CGO_ENABLED=0 go build -o $(BIN) ./cmd/dockerity

test:
	go test ./...

lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "gofmt: files need formatting" && exit 1)

run: build
	DOCKERITY_DATA_DIR=./.data DOCKERITY_LOG_LEVEL=debug ./$(BIN)

clean:
	rm -rf bin
