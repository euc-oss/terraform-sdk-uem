# Workspace ONE UEM Go SDK — consumer Makefile.
#
# This Makefile is for SDK consumers building and testing locally.
# The SDK source itself is generated upstream; this repository ships the
# generated output. Code generation happens upstream; no codegen target exists here.

GO              ?= go
GOLANGCI_LINT   ?= golangci-lint
COVERAGE_OUT    := coverage.out
COVERAGE_HTML   := coverage.html
COVERAGE_MIN    := 50

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile the SDK and all examples
	$(GO) build ./...

.PHONY: test
test: ## Run all tests against the in-process mock server
	TEST_MODE=mock $(GO) test ./...

.PHONY: test-unit
test-unit: ## Run unit tests only (skip integration/)
	TEST_MODE=mock $(GO) test ./client/... ./models/... ./resources/...

.PHONY: test-integration
test-integration: ## Run integration tests only
	TEST_MODE=mock $(GO) test ./tests/...

.PHONY: test-coverage
test-coverage: ## Run tests and emit coverage.out + coverage.html
	TEST_MODE=mock $(GO) test ./... -race -coverprofile=$(COVERAGE_OUT)
	$(GO) tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@coverage=$$($(GO) tool cover -func=$(COVERAGE_OUT) | grep total: | awk '{print $$3}' | sed 's/%//'); \
		echo "Total coverage: $$coverage%"; \
		awk -v c="$$coverage" -v m="$(COVERAGE_MIN)" 'BEGIN{ if (c+0 < m+0) { print "Coverage below minimum (" m "%)"; exit 1 } }'

.PHONY: lint
lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with --fix
	$(GOLANGCI_LINT) run --fix

.PHONY: fmt
fmt: ## Format Go source
	gofmt -w .

.PHONY: quick-check
quick-check: fmt lint test-unit ## Pre-commit check (fmt + lint + unit tests)

.PHONY: ci
ci: lint test-coverage ## Mirror of CI workflow locally

.PHONY: install-tools
install-tools: ## Install pinned developer tooling
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0

.PHONY: clean
clean: ## Remove build / coverage artifacts
	rm -f $(COVERAGE_OUT) $(COVERAGE_HTML)
