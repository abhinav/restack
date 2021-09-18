BIN = bin
GO_FILES = $(shell \
	find . -path '*/.*' -prune -o '(' -type f -a -name '*.go' ')' -print)

RESTACK = $(BIN)/restack

GOLINT = $(BIN)/golint
MOCKGEN = $(BIN)/mockgen
STATICCHECK = $(BIN)/staticcheck

PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
export GOBIN = $(PROJECT_ROOT)/$(BIN)

.PHONY: all
all: build lint test

.PHONY: build
build: $(RESTACK)

.PHONY: test
test: $(GO_FILES)
	go test -race ./...

.PHONY: generate
generate: $(MOCKGEN)
	PATH=$(GOBIN):$$PATH go generate ./...

.PHONY: lint
lint: gofmt golint staticcheck gomodtidy

.PHONY: gofmt
gofmt:
	$(eval FMT_LOG := $(shell mktemp -t gofmt.XXXXX))
	@gofmt -e -s -l $(GO_FILES) > $(FMT_LOG) || true
	@[ ! -s "$(FMT_LOG)" ] || \
		(echo "gofmt failed. Please reformat the following files:" | \
		cat - $(FMT_LOG) && false)

.PHONY: golint
golint: $(GOLINT)
	$(GOLINT) ./...

.PHONY: staticcheck
staticcheck: $(STATICCHECK)
	$(STATICCHECK) ./...

.PHONY: gomodtidy
gomodtidy: go.mod go.sum
	go mod tidy
		@if ! git diff --quiet $^; then \
		echo "go mod tidy changed files:" && \
		git status --porcelain $^ && \
		false; \
	fi

$(RESTACK): $(GO_FILES)
	go install github.com/abhinav/restack/cmd/restack

$(MOCKGEN):
	go install github.com/golang/mock/mockgen

$(GOLINT):
	go install golang.org/x/lint/golint

$(STATICCHECK):
	go install honnef.co/go/tools/cmd/staticcheck
