# docx public-layer Makefile
#
# Drives build / test / lint / coverage / openspec-validate / examples across
# the three Go modules (docx, couchbase, mongo) and the standalone examples.
#
# Run `make help` for a quick command index.

GO          ?= go
PKG_TOP     ?= ./config/... ./utils/... ./observability/...
COVER_FILE  ?= coverage.out
COVER_HTML  ?= coverage.html
CHECK_DEPS  ?= bash scripts/check_deps.sh
EXAMPLES    := $(wildcard example/0*-*)

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "docx Makefile targets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# ----------------------------------------------------------------------
# Build
# ----------------------------------------------------------------------

.PHONY: build
build: ## Build docx public layer
	$(GO) build ./...

.PHONY: build-submodules
build-submodules: ## Build couchbase + mongo sub-modules
	cd couchbase && $(GO) build ./...
	cd mongo     && $(GO) build ./...

.PHONY: build-all
build-all: build build-submodules ## Build everything

# ----------------------------------------------------------------------
# Test
# ----------------------------------------------------------------------

.PHONY: test
test: ## Run docx unit tests (short mode)
	$(GO) test -count=1 -short $(PKG_TOP)

.PHONY: test-submodules
test-submodules: ## Run sub-module unit tests
	cd couchbase && $(GO) test -count=1 -short ./...
	cd mongo     && $(GO) test -count=1 -short ./...

.PHONY: test-all
test-all: test test-submodules ## Run every unit test

.PHONY: test-race
test-race: ## Run docx tests with -race
	$(GO) test -count=1 -race -short $(PKG_TOP)

.PHONY: cover
cover: ## Coverage report (stdout)
	$(GO) test -count=1 -short -cover $(PKG_TOP)

.PHONY: cover-html
cover-html: ## Coverage report (browser HTML)
	$(GO) test -count=1 -short -coverprofile=$(COVER_FILE) $(PKG_TOP)
	$(GO) tool cover -html=$(COVER_FILE) -o $(COVER_HTML)
	@echo "open $(COVER_HTML)"

# ----------------------------------------------------------------------
# Static analysis
# ----------------------------------------------------------------------

.PHONY: vet
vet: ## go vet ./...
	$(GO) vet ./...

.PHONY: deps-check
deps-check: ## Layered dependency check (docx/tracing must not import mqx subpackages or couchbase/mongo)
	$(CHECK_DEPS)

.PHONY: openspec
openspec: ## Validate the active openspec change
	openspec validate --changes --strict

# ----------------------------------------------------------------------
# Examples
# ----------------------------------------------------------------------

.PHONY: build-examples
build-examples: ## Build every example/*
	@for d in $(EXAMPLES); do \
		echo "→ $$d"; \
		( cd $$d && $(GO) build ./... ) || exit 1; \
	done

.PHONY: tidy-examples
tidy-examples: ## go mod tidy every example/*
	@for d in $(EXAMPLES); do \
		echo "→ $$d"; \
		( cd $$d && $(GO) mod tidy ) || exit 1; \
	done

# ----------------------------------------------------------------------
# Convenience
# ----------------------------------------------------------------------

.PHONY: ci
ci: deps-check vet openspec test cover ## Full CI gate (one-shot)
