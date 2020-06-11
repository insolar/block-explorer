export GOPATH ?= $(shell go env GOPATH)
export GO111MODULE ?= on
export GOSUMDB ?= sum.golang.org
export GOFLAGS ?= -mod=vendor
export GOPROXY=https://proxy.golang.org,https://goproxy.io,direct

BIN_DIR = bin
LDFLAGS ?=
COVERPROFILE ?= coverage.out
TEST_LOG_FILENAME ?= autotest.log
ARTIFACTS_DIR = .artifacts

TEST_COUNT ?= 1
TEST_ARGS ?=

#.DEFAULT_GOAL := all

.PHONY: all
all: vendor clean build

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

go-acc: ## install coverage tool
	go get github.com/ory/go-acc@v0.2.3

.PHONY: install_deps
install_deps: golangci go-acc ## install necessary dependencies

.PHONY: build
build:  ## build all applications
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/block-explorer cmd/block-explorer/*.go
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/migrate cmd/migrate/*.go
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/api cmd/api/*.go

.PHONY: vendor
vendor:  ## update vendor dependencies
	go mod vendor

.PHONY: generate
generate: ## generate mocks
	GOFLAGS="" go generate ./...

.PHONY: unit
unit:  ## run unit tests
	go test -v ./... -tags unit -count $(TEST_COUNT) -race $(TEST_ARGS)

.PHONY: test
test: unit integration test-heavy-mock-integration ## run all tests

.PHONY: integration
integration: ## run integrations tests with race
	go test -v ./... -tags integration -count $(TEST_COUNT) -race $(TEST_ARGS)

.PHONY: test-with-coverage
test-with-coverage: ## run tests with coverage mode
	go-acc --covermode=count --output=coverage.tmp.out ./... -- -tags "unit integration heavy_mock_integration" -count=1
	cat coverage.tmp.out | grep -v _mock.go > ${COVERPROFILE}
	go tool cover -html=${COVERPROFILE} -o coverage.html

.PHONY: test-heavy-mock-integration
test-heavy-mock-integration:
	go test -v ./... -tags heavy_mock_integration -count $(TEST_COUNT) -race $(TEST_ARGS) | tee $(TEST_LOG_FILENAME)

.PHONY: publish_tests
publish_tests: ## send results to testrail
	${GOPATH}/bin/testrail-cli --URL=https://insolar.testrail.io/ --USER=$(TR_USER) --PASSWORD=$(TR_PASSWORD) --RUN_ID=108 --FILE=$(TEST_LOG_FILENAME)

.PHONY: lint
lint: ## run linter
	${BIN_DIR}/golangci-lint --color=always run ./... -v --timeout 5m

.PHONY: bench
bench: ## run benchmarks
	go test -v ./... -tags bench -bench=. -benchmem -benchtime=1000x

.PHONY: bench-compare
bench-compare: ## run benchmarks compare for last two commits
	cob -bench-cmd make -bench-args bench -threshold 0.7

.PHONY: bench-integration
bench-integration: ## run integration benchmarks
	go test -v ./... -tags bench_integration -bench=. -benchmem -benchtime=1x

.PHONY: bench-compare-integration
bench-compare-integration: ## run integration benchmarks
	cob -bench-cmd make -bench-args bench-integration -threshold 0.7

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
	protoc -I./vendor -I./ --gogoslick_out=plugins=grpc:./ test/heavymock/import_records.proto

.PHONY: migrate
migrate: ## migrate
	go run ./cmd/migrate/migrate.go --config=.artifacts/migrate.yaml

.PHONY: docker_postgresql
docker_postgresql: ## start docker with postgresql
	./postgresql-docker/doker_run.sh

.PHONY: help
help: ## display help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
