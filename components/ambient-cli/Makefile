BINARY    := acpctl
MODULE    := github.com/ambient-code/platform/components/ambient-cli
PKG       := $(MODULE)/pkg/info

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(PKG).Version=$(VERSION) \
	-X $(PKG).Commit=$(COMMIT) \
	-X $(PKG).BuildDate=$(BUILD_DATE)

.PHONY: build clean fmt vet lint test

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/ambient/

clean:
	rm -f $(BINARY)

fmt:
	gofmt -l -w .

vet:
	go vet ./...

lint: fmt vet
	golangci-lint run

test:
	go test ./...
