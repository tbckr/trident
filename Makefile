TEST_OPTIONS ?=
SOURCE_FILES ?= ./...
TEST_PATTERN ?= .

default: help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build go binary
	go build -o trident cmd/trident/main.go

test: ## Run tests
	go test ${TEST_OPTIONS} -failfast -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.txt ${SOURCE_FILES} -run ${TEST_PATTERN} -timeout=5m

coverage: test ## Generate coverage report
	go tool cover -html=coverage.txt

lint: ## Run linter
	golangci-lint run ./...

tidy: ## Run go mod tidy
	go mod tidy

ci: clean tidy lint test build ## Run CI checks

clean: ## Cleanup
	rm -rf dist
	rm -rf coverage.txt
	rm -rf trident

pre: ## Run pre-commit checks
	pre-commit run --all-files

license: ## Add license headers
	addlicense -c "Tim <tbckr>" -l MIT -s -v \
        -ignore "dist/**" \
        -ignore ".idea/**" \
        -ignore ".task/**" \
        -ignore ".github/licenses.tmpl" \
        -ignore "licenses/*" \
        -ignore "venv/*" \
        .

.PHONY: default help build test coverage lint tidy ci clean pre license