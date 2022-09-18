RELEASE ?=
LLVM_COV_REPORT_FLAGS = --hide-instantiations

ifeq ($(RELEASE),)
BUILD_FLAGS =
else
BUILD_FLAGS = --release
endif

.PHONY: all
all: build lint test

.PHONY: build
build:
	cargo build $(BUILD_FLAGS)

.PHONY: test
test:
	cargo nextest run --workspace

.PHONY: cover
cover:
	cargo llvm-cov nextest --workspace --lcov --output-path lcov.info
	cargo llvm-cov report $(LLVM_COV_REPORT_FLAGS)
	cargo llvm-cov report $(LLVM_COV_REPORT_FLAGS) --html

.PHONY: lint
lint: fmt clippy

.PHONY: fmt
fmt:
	cargo fmt --check

.PHONY: clippy
clippy:
	cargo clippy
