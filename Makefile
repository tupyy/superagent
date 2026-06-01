BINARY := superagent
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)
TAGS := exclude_graphdriver_btrfs containers_image_openpgp

.PHONY: build test lint format clean

build:
	go build -tags "$(TAGS)" -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

test:
	go test -tags "$(TAGS)" ./...

lint:
	golangci-lint run ./...

format:
	gofmt -w .
	goimports -w .

clean:
	rm -rf bin/
