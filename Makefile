# Directory containing the Makefile.
PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

export GOBIN ?= $(PROJECT_ROOT)/bin
export PATH := $(GOBIN):$(PATH)

BIN_DIR         := $(PROJECT_ROOT)/.bin

GOLANGCI_LINT_VERSION = 1.64.6
GOLANGCI_LINT          := $(BIN_DIR)/golangci-lint

.PHONY: all
all: lint test

.PHONY: lint
lint: golangci-lint tidy-lint

$(GOLANGCI_LINT):
	@mkdir -p $(BIN_DIR)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN_DIR) v$(GOLANGCI_LINT_VERSION)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: tidy-lint
tidy-lint:
	go mod tidy
	git diff --exit-code -- go.mod go.sum

.PHONY: test
test:
	go test -race ./...

.PHONY: cover
cover:
	go test -race -coverprofile=cover.out -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html
