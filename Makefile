RELEASE ?= 0
TEST_ARGS ?=

ifeq ($(RELEASE),1)
_BUILD_RELEASE_FLAGS = --release
else
_BUILD_RELEASE_FLAGS =
endif

BUILD_FLAGS = $(_BUILD_RELEASE_FLAGS)

.PHONY: build
build: ## build the binary
	cargo build --locked $(BUILD_FLAGS)

.PHONY: test
test: ## run all tests
	GIT_CONFIG_GLOBAL= cargo nextest run --locked --workspace --no-fail-fast $(TEST_ARGS)

_COV_FLAGS = --hide-instantiations

.PHONY: cover
cover: ## generate a coverage report
	cargo llvm-cov nextest --workspace --locked --lcov --output-path lcov.info --no-fail-fast
	cargo llvm-cov report $(_COV_FLAGS)

.PHONY: cover-html
cover-html: ## generate an HTML coverage report
cover-html: cover
	cargo llvm-cov report $(_COV_FLAGS) --html

.PHONY: fmt
fmt: ## reformat code
	cargo fmt

.PHONY: lint
lint: fmt-check clippy

.PHONY: fmt-check
fmt-check:
	cargo fmt --check

.PHONY: clippy
clippy:
	cargo clippy --workspace
