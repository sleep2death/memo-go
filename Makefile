GO ?= go
GOFMT ?= gofmt "-s"
GO_VERSION=$(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
PACKAGES ?= $(shell $(GO) list ./...)
VETPACKAGES ?= $(shell $(GO) list ./... | grep -v /examples/)
GOFILES := $(shell find . -name "*.go")
COVERFILE=cover.out
TESTTAGS ?= ""

.PHONY: test
test:
	$(GO) test $(TESTTAGS) -covermode=count -coverprofile=$(COVERFILE)

.PHONY: fmt
fmt:
	$(GOFMT) -w $(GOFILES)
