export GOBIN ?= $(CURDIR)/bin

MOCKGEN = $(GOBIN)/mockgen

.PHONY: test
test:
	go test -race -v ./...

.PHONY: generate
generate: $(MOCKGEN)
	PATH=$(GOBIN):$$PATH go generate ./...

$(MOCKGEN):
	go install github.com/golang/mock/mockgen
