BINARY_NAME=dankcal
SOURCE_DIR=cmd/dankcal
BUILD_DIR=bin
PREFIX ?= /usr/local
INSTALL_DIR=$(PREFIX)/bin

GO=go

BASE_VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
COMMIT_COUNT=$(shell git rev-list --count HEAD 2>/dev/null || echo "0")
COMMIT_HASH=$(shell git rev-parse --short=8 HEAD 2>/dev/null || echo "unknown")
VERSION?=$(BASE_VERSION)+git$(COMMIT_COUNT).$(COMMIT_HASH)
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT?=$(COMMIT_HASH)

BUILD_LDFLAGS=-ldflags='-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.Commit=$(COMMIT)'

.PHONY: all build dev install uninstall clean test cover fmt vet deps generate mocks help

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build $(BUILD_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(SOURCE_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

dev:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(SOURCE_DIR)

install:
	@install -D -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

uninstall:
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)

clean:
	@rm -rf $(BUILD_DIR)

test:
	CGO_ENABLED=0 $(GO) test ./...

cover:
	CGO_ENABLED=0 $(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

fmt:
	$(GO) fmt ./...

vet:
	CGO_ENABLED=0 $(GO) vet ./...

deps:
	$(GO) mod tidy
	$(GO) mod download

generate:
	$(GO) generate ./ent/...

mocks:
	mockery

help:
	@echo "Targets: build dev install uninstall clean test cover fmt vet deps generate mocks"
