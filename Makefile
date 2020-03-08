export GOBIN ?= $(CURDIR)/bin

MOCKGEN = $(GOBIN)/mockgen
GOLINT = $(GOBIN)/golint

.PHONY: test
test:
	go test -race -v ./...

.PHONY: lint
lint: $(GOLINT)
	$(GOLINT) ./...

.PHONY: generate
generate: $(MOCKGEN)
	PATH=$(GOBIN):$$PATH go generate ./...

$(MOCKGEN):
	go install github.com/golang/mock/mockgen

$(GOLINT):
	go install golang.org/x/lint/golint
