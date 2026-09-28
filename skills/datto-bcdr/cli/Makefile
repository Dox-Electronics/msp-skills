.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -trimpath -o bin/datto-bcdr-cli$(BIN_EXT) ./cmd/datto-bcdr-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/datto-bcdr-cli

clean:
	rm -rf bin/

build-mcp:
	go build -trimpath -o bin/datto-bcdr-mcp$(BIN_EXT) ./cmd/datto-bcdr-mcp

install-mcp:
	go install ./cmd/datto-bcdr-mcp

build-all: build build-mcp
