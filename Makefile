export GOPATH ?= $(shell go env GOPATH)
export GO111MODULE ?= on

BIN_DIR = bin
LDFLAGS ?=
COVERPROFILE ?= coverage.txt

#.DEFAULT_GOAL := all

.PHONY: all
all: build

.PHONY: mod
mod:
	go mod download

.PHONY: clean
clean: ## run all cleanup tasks
	go clean ./...
	rm -f $(COVERPROFILE)
	rm -rf $(BIN_DIR)

golangci: ## install golangci-linter
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${BIN_DIR} v1.21.0

.PHONY: install_deps
install_deps: golangci ## install necessary dependencies

.PHONY: build
build:  ## build all applications
	@echo build

.PHONY: generate
generate: ## generate mocks
	go generate ./...

.PHONY: unit
unit:  ## run unit tests
	go test -v ./... -count 10 -race

.PHONY: test
test: unit integration ## run unit and integrations tests with race

.PHONY: integration
integration: ## run integrations tests with race
	go test -v ./... -tags integration -count 10 -race --failfast

.PHONY: test-with-coverage
test-with-coverage: ## run tests with coverage mode
	go test -v ./... -tags integration -count 1 --coverprofile=$(COVERPROFILE) --covermode=count

.PHONY: lint
lint: golangci ## run linter
	${BIN_DIR}/golangci-lint --color=always run ./... -v --timeout 5m

.PHONY: help
help: ## display help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
