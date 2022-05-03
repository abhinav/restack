GRCOV_FLAGS = -s . --binary-path ./target/debug --branch --ignore-not-existing
RELEASE ?=

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
	cargo test
# TODO: respect release?

.PHONY: cover
cover: export RUSTFLAGS=-Cinstrument-coverage
cover:
	@rm -f restack-*.profraw lcov.info
	cargo build
	LLVM_PROFILE_FILE=$(shell pwd)/restack-%p-%m.profraw cargo test
	@mkdir -p ./target/debug/coverage
	grcov . $(GRCOV_FLAGS) -t html -o ./target/debug/coverage/
	grcov . $(GRCOV_FLAGS) -t lcov -o lcov.info

.PHONY: lint
lint: fmt clippy

.PHONY: fmt
fmt:
	cargo fmt --check

.PHONY: clippy
clippy:
	cargo clippy
