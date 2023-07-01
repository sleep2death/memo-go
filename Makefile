GO ?= go
SWAG ?= swag
GOFMT ?= gofmt "-s"
GO_VERSION=$(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
PACKAGES ?= $(shell $(GO) list ./...)
VETPACKAGES ?= $(shell $(GO) list ./... | grep -v /examples/)
GOFILES := $(shell find . -name "*.go")
MAINFILE=cmd/main.go
COVERFILE=cover.out
TESTTAGS ?= "./memo"

.PHONY: test
test:
	$(GO) test $(TESTTAGS)

.PHONY: cover
cover:
	$(GO) test $(TESTTAGS) -covermode=count -coverprofile=$(COVERFILE)
	$(GO) tool cover -html=${COVERFILE}

.PHONY: fmt
fmt:
	$(GOFMT) -w $(GOFILES)

.PHONY: docs
docs:
	$(SWAG) init --parseDependency --parseInternal --parseDepth 1 -g $(MAINFILE)
