.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -trimpath -o bin/sentinelone-cli$(BIN_EXT) ./cmd/sentinelone-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/sentinelone-cli

clean:
	rm -rf bin/

build-mcp:
	go build -trimpath -o bin/sentinelone-mcp$(BIN_EXT) ./cmd/sentinelone-mcp

install-mcp:
	go install ./cmd/sentinelone-mcp

build-all: build build-mcp
