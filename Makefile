.PHONY: build test lint run

BINARY := moira

build:
	go build -o bin/$(BINARY) ./cmd/moira

test:
	go test ./... -v -cover

lint:
	golangci-lint run ./...

run: build
	./bin/$(BINARY)