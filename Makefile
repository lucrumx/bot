.PHONY: fmt lint check run-api build

GOLANGCI_LINT := $(CURDIR)/bin/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.64.8

fmt:
	goimports -w -local github.com/lucrumx/bot .
	gofmt -w .

$(GOLANGCI_LINT):
	mkdir -p $(dir $(GOLANGCI_LINT))
	GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: $(GOLANGCI_LINT)
	GOCACHE=$(CURDIR)/.gocache GOLANGCI_LINT_CACHE=$(CURDIR)/.golangci-lint-cache $(GOLANGCI_LINT) run

check: fmt lint

run-api:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api
