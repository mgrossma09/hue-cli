BINARY := ./bin/huectl
MCP_BINARY := ./bin/hue-mcp

.PHONY: test build build-mcp fmt lint

test:
	go test ./...

build:
	mkdir -p ./bin
	go build -o $(BINARY) ./cmd/huectl

build-mcp:
	mkdir -p ./bin
	go build -o $(MCP_BINARY) ./cmd/hue-mcp

fmt:
	gofmt -w $(shell find . -type f -name '*.go' -not -path './vendor/*')

lint:
	golangci-lint run ./...
