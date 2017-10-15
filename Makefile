PACKAGES = $(shell glide nv)

PROJECT_DIR = $(shell pwd)
MOCKGEN     = $(PROJECT_DIR)/.tmp/mockgen
VENDOR      = $(PROJECT_DIR)/vendor

.PHONY: test
test:
	go test -race -v $(PACKAGES)

.PHONY: generate
generate: $(MOCKGEN)
	MOCKGEN=$(MOCKGEN) go generate $(PACKAGES)

$(MOCKGEN):
	@mkdir -p $(dir $(MOCKGEN))
	@echo "Building mockgen"; \
		DIR=$$(mktemp -d) && \
		cd $$DIR && \
		ln -s $(VENDOR) $$DIR/src && \
		GOPATH=$$DIR go build -o $(MOCKGEN) github.com/golang/mock/mockgen && \
		rm -rf $$DIR
