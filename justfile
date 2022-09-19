# list all recipes
default:
	@just --list

_release := env_var_or_default("RELEASE", "0")
_build_flags := if _release == "1" { "--release" } else { "" }

# build the binary
build:
	cargo build {{_build_flags}}

# run all tests
test *args:
	cargo nextest run --workspace --no-fail-fast {{args}}

_llvm_cov_report_flags := "--hide-instantiations"

# generate a coverage report
cover:
	cargo llvm-cov nextest --workspace --lcov --output-path lcov.info --no-fail-fast
	cargo llvm-cov report {{_llvm_cov_report_flags}}

# generate an HTML coverage report
cover-html: cover
	cargo llvm-cov report {{_llvm_cov_report_flags}} --html

# reformat code
fmt:
	cargo fmt

# run linters
lint: _fmt _clippy

_fmt:
	cargo fmt --check

_clippy:
	cargo clippy
