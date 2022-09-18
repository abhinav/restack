GRCOV ?= grcov
RELEASE ?=
GRCOV_FLAGS = \
	--source-dir . \
	--binary-path ./target/debug \
	--branch \
	--ignore-not-existing \
	--ignore "**/tests/*" \
	--excl-start '^mod tests \{' --excl-stop '^\}'
TEST_FLAGS = --features 'anyhow/backtrace'

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
	RUST_BACKTRACE=1 cargo test $(TEST_FLAGS)
# TODO: respect release?

.PHONY: cover
cover: export RUSTFLAGS=-Cinstrument-coverage
cover:
	@rm -f restack-*.profraw lcov.info
	cargo build --tests
	RUST_BACKTRACE=1 LLVM_PROFILE_FILE=$(shell pwd)/restack-%p-%m.profraw \
		       cargo test $(TEST_FLAGS)
	@mkdir -p ./target/debug/coverage
	$(GRCOV) . $(GRCOV_FLAGS) -t html -o ./target/debug/coverage/
	$(GRCOV) . $(GRCOV_FLAGS) -t lcov -o lcov.info

.PHONY: lint
lint: fmt clippy

.PHONY: fmt
fmt:
	cargo fmt --check

.PHONY: clippy
clippy:
	cargo clippy
