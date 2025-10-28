# Makefile for msync

.PHONY: build clean test install deps fmt lint

# Variables
BINARY_NAME=msync
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "1.0.0")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run benchmarks
bench:
	go test -bench=. ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Install binary
install: build
	install -m 755 $(BINARY_NAME) /usr/local/bin/

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-* coverage.out coverage.html

# Development setup
dev-setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run integration tests
integration-test: build
	./scripts/integration_test.sh

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  build-all       - Build for multiple platforms"
	@echo "  test            - Run unit tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  bench           - Run benchmarks"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  deps            - Install dependencies"
	@echo "  install         - Install binary to /usr/local/bin"
	@echo "  clean           - Clean build artifacts"
	@echo "  dev-setup       - Set up development tools"
	@echo "  help            - Show this help"