export GOPATH ?= $(shell go env GOPATH)
export GO111MODULE ?= on

BIN_DIR = bin
LDFLAGS ?=
COVERPROFILE ?= coverage.txt
ARTIFACTS_DIR = .artifacts

#.DEFAULT_GOAL := all

.PHONY: all
all: clean vendor build

.PHONY: mod
mod:
	go mod download

.PHONY: clean
clean: ## run all cleanup tasks
	go clean ./...
	rm -f $(COVERPROFILE)
	rm -rf $(BIN_DIR)

golangci: ## install golangci-linter
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${BIN_DIR} v1.27.0

.PHONY: install_deps
install_deps: golangci ## install necessary dependencies

.PHONY: build
build:  ## build all applications
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/block-explorer cmd/block-explorer/*.go

.PHONY: vendor
vendor:  ## update vendor dependencies
	go mod vendor

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
	go test -v ./... -tags integration -count 10 -race

.PHONY: test-with-coverage
test-with-coverage: ## run tests with coverage mode
	go test -v ./... -tags integration -count 1 --coverprofile=$(COVERPROFILE) --covermode=count

.PHONY: lint
lint: golangci ## run linter
	${BIN_DIR}/golangci-lint --color=always run ./... -v --timeout 5m

.PHONY: config
config: ## generate config
	mkdir -p $(ARTIFACTS_DIR)
	go run ./configuration/gen/gen.go

.PHONY: generate-protobuf
generate-protobuf: ## generate protobuf structs
	@ if ! which protoc > /dev/null; then \
		echo "error: protoc not installed" >&2; \
		exit 1; \
	fi
	protoc --gogoslick_out=plugins=grpc:./ etl/connection/testdata/helloworld.proto

.PHONY: migrate
migrate: ## migrate
	go run ./cmd/migrate/migrate.go --config=.artifacts/migrate.yaml

.PHONY: docker_postgresql
docker_postgresql: ## start docker with postgresql
	./postgresql-docker/doker_run.sh

.PHONY: help
help: ## display help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
