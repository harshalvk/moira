.PHONY: setup build test lint fmt vet tidy hooks kind-up kind-down dev run ci

BINARY := moira

setup:
	go mod download
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/evilmartians/lefthook@latest
	lefthook install
	@echo "setup complete — run 'make kind-up' then 'make dev'"

build:
	go build -o bin/$(BINARY) ./cmd/moira

test:
	go test ./... -v -race -cover

lint:
	golangci-lint run ./...

fmt:
	gofmt -l -w .
	goimports -l -w .

vet:
	go vet ./...

tidy:
	go mod tidy

hooks:
	lefthook install

kind-up:
	./hack/kind-cluster.sh

kind-down:
	./hack/kind-cluster-down.sh

dev:
	air

run: build
	./bin/$(BINARY)

ci: fmt vet lint test build
