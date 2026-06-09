BINARY  := hepler
DIST    := dist
GO      ?= go

# Version comes from git (tag if present, else short SHA); -dirty if uncommitted.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# Release targets (GOOS/GOARCH). hepler targets macOS and Linux only.
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: all build install run test vet fmt tidy clean release version help

all: build

## build: compile hepler for the host platform
build:
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) .

## install: install hepler into $GOBIN / $GOPATH/bin
install:
	$(GO) install -trimpath -ldflags '$(LDFLAGS)' .

## test: run the test suite
test:
	$(GO) test ./...

## vet: run go vet
vet:
	$(GO) vet ./...

## fmt: format the source
fmt:
	$(GO) fmt ./...

## tidy: tidy go.mod
tidy:
	$(GO) mod tidy

## clean: remove build artifacts
clean:
	rm -rf $(BINARY) $(DIST)

## release: cross-compile, archive, and checksum all platforms into dist/
release: test vet
	rm -rf $(DIST) && mkdir -p $(DIST)
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		name=$(BINARY)_$(VERSION)_$${os}_$${arch}; \
		echo ">> $$name"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
			$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY) . || exit 1; \
		tar -C $(DIST) -czf $(DIST)/$$name.tar.gz $(BINARY) || exit 1; \
		rm -f $(DIST)/$(BINARY); \
	done
	@cd $(DIST) && (shasum -a 256 *.tar.gz 2>/dev/null || sha256sum *.tar.gz) > SHA256SUMS
	@echo "Built release $(VERSION) in $(DIST)/"

## version: print the version that would be embedded
version:
	@echo $(VERSION)

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
