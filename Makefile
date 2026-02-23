.PHONY: build test lint clean

# Build variables
BINARY_NAME := hoofy
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := bin
LDFLAGS := -ldflags "-X github.com/HendryAvila/Hoofy/internal/server.Version=$(VERSION)"

## build: Compile the binary
build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/hoofy/

## test: Run all tests
test:
	go test -race -cover ./...

## lint: Run linters (golangci-lint)
lint:
	golangci-lint run ./...

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## tidy: Clean up go.mod and go.sum
tidy:
	go mod tidy

## run: Build and run the MCP server
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) serve

## help: Show this help message
help:
	@printf "Usage: make [target]\n\nTargets:\n"
	@grep -E '^## ' Makefile | sed 's/## /  /' | sed 's/: /\t/'
