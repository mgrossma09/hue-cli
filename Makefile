BINARY := ./bin/huectl

.PHONY: test build fmt lint

test:
	go test ./...

build:
	mkdir -p ./bin
	go build -o $(BINARY) ./cmd/huectl

fmt:
	gofmt -w $(shell find . -type f -name '*.go' -not -path './vendor/*')

lint:
	golangci-lint run ./...
