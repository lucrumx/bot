.PHONY: fmt lint check run-api build

fmt:
	goimports -w -local github.com/lucrumx/bot .
	gofmt -w .

lint:
	golangci-lint run

check: fmt lint

run-api:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api